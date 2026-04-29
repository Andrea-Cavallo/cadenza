package generator

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/Andrea-Cavallo/cadenza/internal/cache"
	"github.com/Andrea-Cavallo/cadenza/internal/llm"
	midipkg "github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

type Config struct {
	BPM       float64
	Key       theory.Key
	Provider  string
	Model     string
	NoLLM     bool
	OutputDir string
	Timestamp string
}

type PatternResult struct {
	Path      string
	NoteCount int
}

type GenerationResult struct {
	Files         []string
	VariationSeed string
	BPM           float64
	Key           string
}

type patternResult struct {
	patternType string
	spec        *schema.PatternSpec
	err         error
}

type MultiGenerator struct {
	provider  llm.Provider
	validator *schema.Validator
	registry  *styleprofile.Registry
	renderer  *renderer.Renderer
	writer    *midipkg.Writer
	cache     *cache.Cache
	outputDir string
	NoLLM     bool
}

func NewMultiGenerator(
	provider llm.Provider,
	validator *schema.Validator,
	registry *styleprofile.Registry,
	rend *renderer.Renderer,
	writer *midipkg.Writer,
	outputDir string,
) *MultiGenerator {
	return &MultiGenerator{
		provider:  provider,
		validator: validator,
		registry:  registry,
		renderer:  rend,
		writer:    writer,
		cache:     cache.New(".cache"),
		outputDir: outputDir,
	}
}

func (mg *MultiGenerator) Generate(ctx context.Context, bpm float64, keyStr string) (*GenerationResult, error) {
	parsedKey, err := theory.ParseKey(keyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid key %q: %w", keyStr, err)
	}

	if bpm < 80 || bpm > 150 {
		return nil, fmt.Errorf("bpm must be between 80-150, got %.1f", bpm)
	}

	seed := NewVariationSeed()
	prog := theory.SelectProgression(parsedKey.Root, parsedKey.Scale, seed)

	slog.Info("chord progression",
		"key", parsedKey.Root+" "+parsedKey.Mode,
		"chords", progressionString(prog),
	)

	musicCtx := MusicContext{
		BPM:              bpm,
		Key:              parsedKey,
		Bars:             16,
		VariationSeed:    seed,
		ChordProgression: prog,
	}

	patternTypes := []string{"bassline", "arpeggio", "melody"}
	results := make(chan patternResult, len(patternTypes))
	var wg sync.WaitGroup

	for _, pt := range patternTypes {
		wg.Add(1)
		go func(pType string) {
			defer wg.Done()
			spec, err := mg.generatePattern(ctx, musicCtx, pType)
			results <- patternResult{patternType: pType, spec: spec, err: err}
		}(pt)
	}

	wg.Wait()
	close(results)

	patterns := make(map[string]*schema.PatternSpec)
	for r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("%s: %w", r.patternType, r.err)
		}
		patterns[r.patternType] = r.spec
	}

	var outputFiles []string
	for _, pt := range patternTypes {
		spec := patterns[pt]
		profile, err := mg.registry.LoadForType(pt, spec.StyleProfile)
		if err != nil {
			profile, err = mg.registry.DefaultForType(pt)
			if err != nil {
				return nil, fmt.Errorf("%s profile: %w", pt, err)
			}
		}

		events, err := mg.renderer.Render(spec, profile)
		if err != nil {
			return nil, fmt.Errorf("%s render: %w", pt, err)
		}

		filename := fmt.Sprintf("%s_%s_%s_%.0f.mid", "output", pt, keyStr, bpm)
		outputPath := filepath.Join(mg.outputDir, filename)

		if err := mg.writer.WriteFile(outputPath, events); err != nil {
			return nil, fmt.Errorf("%s write: %w", pt, err)
		}

		slog.Info("wrote MIDI", "file", outputPath, "events", len(events))
		outputFiles = append(outputFiles, outputPath)
	}

	return &GenerationResult{
		Files:         outputFiles,
		VariationSeed: seed,
		BPM:           bpm,
		Key:           keyStr,
	}, nil
}

func (mg *MultiGenerator) generatePattern(ctx context.Context, musicCtx MusicContext, patternType string) (*schema.PatternSpec, error) {
	if mg.NoLLM {
		spec := offlineTemplate(patternType, musicCtx)
		if spec == nil {
			return nil, fmt.Errorf("no offline template for %q", patternType)
		}
		return spec, nil
	}

	gen := NewGenerator(mg.provider, mg.validator, mg.cache)
	spec, err := gen.Generate(ctx, musicCtx, patternType)
	if err != nil {
		slog.Warn("LLM generation failed, falling back to offline template",
			"type", patternType,
			"error", err,
		)
		spec = offlineTemplate(patternType, musicCtx)
		if spec == nil {
			return nil, fmt.Errorf("no offline template for %q after LLM failure: %w", patternType, err)
		}
	}
	return spec, nil
}

// GenerateAll is the top-level entry point called from main.go.
func GenerateAll(ctx context.Context, cfg Config) ([]PatternResult, error) {
	provider, err := buildProvider(cfg)
	if err != nil {
		return nil, err
	}

	v := schema.NewValidator()
	reg := styleprofile.NewRegistry()
	rend := renderer.New()
	w := midipkg.NewWriter(cfg.BPM)

	mg := NewMultiGenerator(provider, v, reg, rend, w, cfg.OutputDir)
	mg.NoLLM = cfg.NoLLM

	result, err := mg.Generate(ctx, cfg.BPM, cfg.Key.Root+modeFlag(cfg.Key))
	if err != nil {
		return nil, err
	}

	var out []PatternResult
	for _, f := range result.Files {
		out = append(out, PatternResult{Path: f, NoteCount: 0})
	}
	return out, nil
}

func buildProvider(cfg Config) (llm.Provider, error) {
	if cfg.NoLLM {
		return &llm.MockProvider{}, nil
	}
	switch cfg.Provider {
	case "claude":
		return llm.NewClaudeProvider(cfg.Model)
	case "ollama":
		return llm.NewOllamaProvider("http://localhost:11434", cfg.Model), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", cfg.Provider)
	}
}

func modeFlag(k theory.Key) string {
	if k.Mode == "minor" {
		return "m"
	}
	return ""
}
