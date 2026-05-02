package generator

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/Andrea-Cavallo/cadenza/internal/cache"
	"github.com/Andrea-Cavallo/cadenza/internal/llm"
	"github.com/Andrea-Cavallo/cadenza/internal/metrics"
	midipkg "github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// Config holds generation parameters for the GenerateAll entry point.
type Config struct {
	BPM       float64
	Key       theory.Key
	Provider  string
	Model     string
	NoLLM     bool
	OutputDir string
	Timestamp string
}

// PatternResult holds the output file path and note count for a single pattern.
type PatternResult struct {
	Path      string
	NoteCount int
}

// GenerationResult is the output of a full 3-track generation session.
type GenerationResult struct {
	Files         []string
	VariationSeed string
	BPM           float64
	Key           string
	Specs         map[string]*schema.PatternSpec
}

type patternResult struct {
	patternType string
	spec        *schema.PatternSpec
	err         error
}

// MultiGenerator orchestrates parallel generation of 3 pattern types with fallback.
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
		cache:     cache.New(30, ".cache"), // 30-day TTL
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
		Groove:           "straight",
	}

	return mg.generateInternal(ctx, musicCtx, 1)
}

// GenerateWithContext generates patterns with full context control (supports all new flags)
func (mg *MultiGenerator) GenerateWithContext(ctx context.Context, musicCtx MusicContext, bars, varNum int) (*GenerationResult, error) {
	return mg.generateInternal(ctx, musicCtx, varNum)
}

// GeneratePartWithContext generates and renders a single pattern type while
// preserving the same musical context used for a full three-part session.
func (mg *MultiGenerator) GeneratePartWithContext(ctx context.Context, musicCtx MusicContext, patternType string, varNum int) (*GenerationResult, error) {
	if !isKnownPatternType(patternType) {
		return nil, fmt.Errorf("unknown pattern type %q", patternType)
	}

	spec, err := mg.generatePattern(ctx, musicCtx, patternType)
	if err != nil {
		return nil, err
	}

	ts := time.Now().Format("20060102_150405")
	keyStr := musicCtx.Key.Root
	if musicCtx.Key.Mode == "minor" {
		keyStr += "m"
	}
	path, err := mg.renderAndSave(patternType, spec, keyStr, ts, varNum, musicCtx.BPM)
	if err != nil {
		return nil, err
	}

	return &GenerationResult{
		Files:         []string{path},
		VariationSeed: musicCtx.VariationSeed,
		BPM:           musicCtx.BPM,
		Key:           keyStr,
		Specs:         map[string]*schema.PatternSpec{patternType: spec},
	}, nil
}

func isKnownPatternType(patternType string) bool {
	return patternType == "bassline" || patternType == "arpeggio" || patternType == "melody"
}

func (mg *MultiGenerator) generateInternal(ctx context.Context, musicCtx MusicContext, varNum int) (*GenerationResult, error) {
	metrics.GenerationsTotal.Add(1)
	ts := time.Now().Format("20060102_150405")

	slog.Debug("generation starting",
		"variation", varNum,
		"key", musicCtx.Key.Root+" "+musicCtx.Key.Mode,
		"bpm", musicCtx.BPM,
		"bars", musicCtx.Bars,
		"groove", musicCtx.Groove,
		"seed", musicCtx.VariationSeed,
		"no_llm", mg.NoLLM,
	)

	patternTypes := []string{"bassline", "arpeggio", "melody"}
	results := make(chan patternResult, len(patternTypes))
	var wg sync.WaitGroup

	for _, pt := range patternTypes {
		wg.Add(1)
		go func(pType string) {
			defer wg.Done()
			slog.Debug("generating pattern", "type", pType, "seed", musicCtx.VariationSeed)
			spec, err := mg.generatePattern(ctx, musicCtx, pType)
			if err != nil {
				slog.Debug("pattern generation failed", "type", pType, "error", err)
			} else {
				slog.Debug("pattern generated", "type", pType, "steps", len(spec.Motif.Steps), "profile", spec.StyleProfile)
			}
			results <- patternResult{patternType: pType, spec: spec, err: err}
		}(pt)
	}

	wg.Wait()
	close(results)

	patterns := make(map[string]*schema.PatternSpec)
	for r := range results {
		if r.err != nil {
			metrics.GenerationErrors.Add(1)
			return nil, fmt.Errorf("%s: %w", r.patternType, r.err)
		}
		patterns[r.patternType] = r.spec
	}
	arrangementScore := schema.ScoreArrangement(patterns)
	slog.Info("arrangement score",
		"peak_section_by_track", arrangementScore.PeakSectionByTrack,
		"all_tracks_peak_same_section", arrangementScore.AllTracksPeakSameSection,
		"melody_arp_pitch_collision", arrangementScore.MelodyArpPitchCollision,
		"melody_arp_register_collision", arrangementScore.MelodyArpRegisterCollision,
		"warnings", arrangementScore.Warnings,
	)

	var outputFiles []string
	keyStr := musicCtx.Key.Root
	if musicCtx.Key.Mode == "minor" {
		keyStr += "m"
	}
	for _, pt := range patternTypes {
		path, err := mg.renderAndSave(pt, patterns[pt], keyStr, ts, varNum, musicCtx.BPM)
		if err != nil {
			return nil, err
		}
		outputFiles = append(outputFiles, path)
	}

	return &GenerationResult{
		Files:         outputFiles,
		VariationSeed: musicCtx.VariationSeed,
		BPM:           musicCtx.BPM,
		Key:           keyStr,
		Specs:         patterns,
	}, nil
}

// renderAndSave renders a single pattern spec and writes the MIDI file. Returns the output path.
func (mg *MultiGenerator) renderAndSave(pt string, spec *schema.PatternSpec, keyStr, ts string, varNum int, bpm float64) (string, error) {
	profile, err := mg.registry.LoadForType(pt, spec.StyleProfile)
	if err != nil {
		slog.Debug("custom profile not found, using default", "type", pt, "requested", spec.StyleProfile, "error", err)
		profile, err = mg.registry.DefaultForType(pt)
		if err != nil {
			return "", fmt.Errorf("%s profile: %w", pt, err)
		}
	}
	slog.Debug("rendering with profile", "type", pt, "profile", profile.Name, "bars", spec.Meta.Bars)

	events, err := mg.renderer.Render(spec, profile)
	if err != nil {
		return "", fmt.Errorf("%s render: %w", pt, err)
	}
	slog.Debug("rendered", "type", pt, "events", len(events))

	suffix := ""
	if varNum > 1 {
		suffix = fmt.Sprintf("_v%d", varNum)
	}
	filename := fmt.Sprintf("%s_%s_%s_%.0f_%s%s.mid", "output", pt, keyStr, bpm, ts, suffix)
	outputPath := filepath.Join(mg.outputDir, filename)

	if err := mg.writer.WriteFile(outputPath, events); err != nil {
		return "", fmt.Errorf("%s write: %w", pt, err)
	}
	slog.Info("wrote MIDI", "file", outputPath, "events", len(events))
	return outputPath, nil
}

func (mg *MultiGenerator) generatePattern(ctx context.Context, musicCtx MusicContext, patternType string) (*schema.PatternSpec, error) {
	if mg.NoLLM {
		metrics.OfflinePatterns.Add(1)
		spec := offlineTemplate(patternType, musicCtx)
		if spec == nil {
			return nil, fmt.Errorf("no offline template for %q", patternType)
		}
		return spec, nil
	}

	metrics.LLMCalls.Add(1)
	gen := NewGenerator(mg.provider, mg.validator, mg.cache)
	spec, err := gen.Generate(ctx, musicCtx, patternType)
	if err != nil {
		metrics.LLMErrors.Add(1)
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
