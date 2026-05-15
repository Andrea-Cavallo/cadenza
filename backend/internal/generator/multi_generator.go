package generator

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

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
	BPM            float64
	Key            theory.Key
	Provider       string
	Model          string
	NoLLM          bool
	OutputDir      string
	Timestamp      string
	Temperature    float64 // 0.0 = use default (0.3)
	MaxRetries     int     // 0 = use default (3)
	TimeoutSeconds int     // 0 = use default (30)
}

// PatternResult holds the output file path and note count for a single pattern.
type PatternResult struct {
	Path      string
	NoteCount int
}

// GenerationResult is the output of a full generation session (3 or 7 stems).
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
	provider       llm.Provider
	validator      *schema.Validator
	registry       *styleprofile.Registry
	renderer       *renderer.Renderer
	writer         *midipkg.Writer
	cache          *cache.Cache
	outputDir      string
	NoLLM          bool
	Sequential     bool // run patterns one-at-a-time (needed for local Ollama which queues requests)
	temperature    float64
	maxRetries     int
	timeoutSeconds int
	warnMu         sync.Mutex
	warnings       []string
}

func (mg *MultiGenerator) addWarning(msg string) {
	mg.warnMu.Lock()
	defer mg.warnMu.Unlock()
	mg.warnings = append(mg.warnings, msg)
}

// Warnings returns any LLM-fallback warnings accumulated during the last generation.
func (mg *MultiGenerator) Warnings() []string {
	mg.warnMu.Lock()
	defer mg.warnMu.Unlock()
	return append([]string(nil), mg.warnings...)
}

func (mg *MultiGenerator) clearWarnings() {
	mg.warnMu.Lock()
	defer mg.warnMu.Unlock()
	mg.warnings = mg.warnings[:0]
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
		provider:       provider,
		validator:      validator,
		registry:       registry,
		renderer:       rend,
		writer:         writer,
		cache:          cache.New(30, ".cache"), // 30-day TTL
		outputDir:      outputDir,
		temperature:    0.3,
		maxRetries:     3,
		timeoutSeconds: 30,
	}
}

// SetLLMOverrides allows setting temperature, max retries, and timeout from config.
func (mg *MultiGenerator) SetLLMOverrides(temperature float64, maxRetries, timeoutSeconds int) {
	if temperature > 0 {
		mg.temperature = temperature
	}
	if maxRetries > 0 {
		mg.maxRetries = maxRetries
	}
	if timeoutSeconds > 0 {
		mg.timeoutSeconds = timeoutSeconds
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

	keyStr := musicCtx.Key.Root
	if musicCtx.Key.Mode == "minor" {
		keyStr += "m"
	}
	path, err := mg.renderAndSave(patternType, spec, keyStr, musicCtx.VariationSeed, varNum, musicCtx.BPM)
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
	switch patternType {
	case "bassline", "bassline_rolling", "bassline_sub",
		"arpeggio", "melody", "chord_pad", "lead_stab":
		return true
	}
	return false
}

// patternTypesForContext returns the ordered list of pattern types to generate.
// Offline mode with StyleFamily set → 7 stems. LLM mode → 3 stems (bass/arp/melody).
func patternTypesForContext(musicCtx MusicContext, noLLM bool) []string {
	if noLLM && musicCtx.StyleFamily != "" {
		return []string{
			"bassline", "bassline_rolling", "bassline_sub",
			"arpeggio", "melody", "chord_pad", "lead_stab",
		}
	}
	return []string{"bassline", "arpeggio", "melody"}
}

func (mg *MultiGenerator) generateInternal(ctx context.Context, musicCtx MusicContext, varNum int) (*GenerationResult, error) {
	mg.clearWarnings()
	metrics.GenerationsTotal.Add(1)

	slog.Debug("generation starting",
		"variation", varNum,
		"key", musicCtx.Key.Root+" "+musicCtx.Key.Mode,
		"bpm", musicCtx.BPM,
		"bars", musicCtx.Bars,
		"groove", musicCtx.Groove,
		"style_family", string(musicCtx.StyleFamily),
		"seed", musicCtx.VariationSeed,
		"no_llm", mg.NoLLM,
	)

	patternTypes := patternTypesForContext(musicCtx, mg.NoLLM)
	patterns := make(map[string]*schema.PatternSpec, len(patternTypes))

	if mg.Sequential {
		// Sequential mode: one request at a time (required for Ollama which queues all requests).
		for _, pt := range patternTypes {
			slog.Debug("generating pattern", "type", pt, "seed", musicCtx.VariationSeed)
			spec, err := mg.generatePattern(ctx, musicCtx, pt)
			if err != nil {
				metrics.GenerationErrors.Add(1)
				return nil, fmt.Errorf("%s: %w", pt, err)
			}
			slog.Debug("pattern generated", "type", pt, "steps", len(spec.Motif.Steps), "profile", spec.StyleProfile)
			patterns[pt] = spec
		}
	} else {
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

		for r := range results {
			if r.err != nil {
				metrics.GenerationErrors.Add(1)
				return nil, fmt.Errorf("%s: %w", r.patternType, r.err)
			}
			patterns[r.patternType] = r.spec
		}
	}

	// Arrangement scoring only applies to the core 3-track set.
	corePatterns := map[string]*schema.PatternSpec{}
	for _, k := range []string{"bassline", "arpeggio", "melody"} {
		if p, ok := patterns[k]; ok {
			corePatterns[k] = p
		}
	}
	if len(corePatterns) == 3 {
		arrangementScore := schema.ScoreArrangement(corePatterns)
		slog.Info("arrangement score",
			"peak_section_by_track", arrangementScore.PeakSectionByTrack,
			"all_tracks_peak_same_section", arrangementScore.AllTracksPeakSameSection,
			"melody_arp_pitch_collision", arrangementScore.MelodyArpPitchCollision,
			"melody_arp_register_collision", arrangementScore.MelodyArpRegisterCollision,
			"warnings", arrangementScore.Warnings,
		)
	}

	var outputFiles []string
	keyStr := musicCtx.Key.Root
	if musicCtx.Key.Mode == "minor" {
		keyStr += "m"
	}
	for _, pt := range patternTypes {
		path, err := mg.renderAndSaveNamed(pt, patterns[pt], keyStr, musicCtx.VariationSeed, varNum, musicCtx.BPM)
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
func (mg *MultiGenerator) renderAndSave(pt string, spec *schema.PatternSpec, keyStr, seed string, varNum int, bpm float64) (string, error) {
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

	filename := midiFilename(pt, keyStr, bpm, varNum, seed)
	outputPath := filepath.Join(mg.outputDir, filename)

	if err := mg.writer.WriteFile(outputPath, events); err != nil {
		return "", fmt.Errorf("%s write: %w", pt, err)
	}
	slog.Info("wrote MIDI", "file", outputPath, "events", len(events))
	return outputPath, nil
}

// stemName maps internal pattern type identifiers to human-readable stem file names.
func stemName(patternType string) string {
	switch patternType {
	case "bassline":
		return "bassline-groove"
	case "bassline_rolling":
		return "bassline-rolling"
	case "bassline_sub":
		return "bassline-sub"
	case "arpeggio":
		return "arp"
	case "melody":
		return "melody"
	case "chord_pad":
		return "chord-pad"
	case "lead_stab":
		return "lead"
	default:
		return patternType
	}
}

// renderAndSaveNamed is like renderAndSave but uses stem-specific file names for 7-stem bundles.
func (mg *MultiGenerator) renderAndSaveNamed(pt string, spec *schema.PatternSpec, keyStr, seed string, varNum int, bpm float64) (string, error) {
	profile, err := mg.registry.LoadForType(pt, spec.StyleProfile)
	if err != nil {
		slog.Debug("custom profile not found, using default", "type", pt, "requested", spec.StyleProfile, "error", err)
		profile, err = mg.registry.DefaultForTypeExtended(pt)
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

	filename := midiFilename(stemName(pt), keyStr, bpm, varNum, seed)
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
	gen.SetLLMParams(mg.temperature, mg.maxRetries)
	spec, err := gen.Generate(ctx, musicCtx, patternType)
	if err != nil {
		metrics.LLMErrors.Add(1)
		slog.Warn("LLM generation failed, falling back to offline template",
			"type", patternType,
			"error", err,
		)
		mg.addWarning(fmt.Sprintf("LLM failed for %s — offline template used (%v)", patternType, err))
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
	case "openai":
		return llm.NewOpenAIProvider(cfg.Model)
	case "gemini":
		return llm.NewGeminiProvider(cfg.Model)
	case "deepseek":
		return llm.NewDeepSeekProvider(cfg.Model)
	case "groq":
		return llm.NewGroqProvider(cfg.Model)
	case "mistral":
		return llm.NewMistralProvider(cfg.Model)
	default:
		return nil, fmt.Errorf("unknown provider %q", cfg.Provider)
	}
}

// midiFilename builds a clean, reproducible filename:
// cadenza_bass_Am_122_v1_s847261.mid
func midiFilename(stem, key string, bpm float64, varNum int, seed string) string {
	shortSeed := seed
	if len(shortSeed) > 6 {
		shortSeed = shortSeed[:6]
	}
	return fmt.Sprintf("cadenza_%s_%s_%.0f_v%d_s%s.mid", stem, key, bpm, varNum, shortSeed)
}

func modeFlag(k theory.Key) string {
	if k.Mode == "minor" {
		return "m"
	}
	return ""
}
