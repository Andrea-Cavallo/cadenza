package generator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Andrea-Cavallo/cadenza/internal/cache"
	"github.com/Andrea-Cavallo/cadenza/internal/llm"
	embeddedprompts "github.com/Andrea-Cavallo/cadenza/internal/prompts"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// StyleFamily drives adaptive generation across all layers when provider=offline.
type StyleFamily string

const (
	StyleFamilyGroove  StyleFamily = "groove"
	StyleFamilyRolling StyleFamily = "rolling"
	StyleFamilySub     StyleFamily = "sub"
)

// MusicContext carries all musical parameters for a generation pass.
type MusicContext struct {
	BPM              float64
	Key              theory.Key
	Bars             int
	VariationSeed    string
	ChordProgression theory.ChordProgression
	Groove           string      // timing preset: straight, mpc60, linndrum, humanize
	OfflineStyle     string      // melodic, hypnotic, driving, minimal (legacy — use StyleFamily)
	StyleFamily      StyleFamily // groove, rolling, sub — applies only when provider=offline
}

// SingleGenerator handles one pattern type generation via LLM with caching.
type SingleGenerator struct {
	provider  llm.Provider
	validator *schema.Validator
	cache     *cache.Cache
	promptDir string
}

func NewGenerator(provider llm.Provider, validator *schema.Validator, c *cache.Cache) *SingleGenerator {
	return &SingleGenerator{
		provider:  provider,
		validator: validator,
		cache:     c,
	}
}

const systemPrompt = `You are an expert electronic music producer specializing in progressive house and melodic techno.
You generate musical patterns as structured JSON matching the PatternSpec schema.

Rules:
- All notes MUST be from the specified scale
- Respect the MIDI note range for the pattern type
- Follow the chord progression: use chord tones as primary notes in each section
- Evolution phases must cover all 16 bars without gaps or overlaps
- Density must stay within the specified range for the pattern type
- variation_seed must match the provided seed exactly
- Return ONLY valid JSON, no markdown, no explanation

CRITICAL — evolution field constraints (violations cause rejection):
- "action" must be exactly one of these strings (no other values accepted):
    introduce | build | peak | release | octave_up | octave_down |
    density_up | density_down | add_chord_note | strip_to_root | ornament
- "intensity" must be a decimal float in [0.0, 1.0] — e.g. 0.3 or 0.8. Never an integer like 3 or 10.
- A typical 4-phase evolution: introduce(0.3) → build(0.6) → peak(0.9) → release(0.5)`

func (g *SingleGenerator) readPrompt(patternType string) ([]byte, error) {
	if g.promptDir != "" {
		path := filepath.Join(g.promptDir, patternType+"_v1.md")
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read prompt %q: %w", path, err)
		}
		return data, nil
	}
	data, err := embeddedprompts.FS.ReadFile(patternType + "_v1.md")
	if err != nil {
		return nil, fmt.Errorf("read prompt %q: %w", patternType+"_v1.md", err)
	}
	return data, nil
}

func (g *SingleGenerator) Generate(ctx context.Context, musicCtx MusicContext, patternType string) (*schema.PatternSpec, error) {
	promptTemplate, err := g.readPrompt(patternType)
	if err != nil {
		return nil, err
	}

	plan := g.loadOrBuildMusicalPlan(musicCtx, patternType)

	// REFACTOR.md point 6: Include prompt template hash in cache key for invalidation on prompt changes
	promptHash := hashContent(promptTemplate)
	cacheKeys := []string{
		g.provider.Name(),
		patternType,
		musicCtx.Key.Root,
		musicCtx.Key.Mode,
		musicCtx.VariationSeed,
		promptHash,
		plannerVersion,
		styleCardVersion,
		plan.StyleCard.Name,
		criticVersion,
		revisionPolicyVersion,
	}

	if g.cache != nil {
		if cached, ok := g.cache.Get(cacheKeys...); ok {
			var spec schema.PatternSpec
			if err := json.Unmarshal(cached, &spec); err == nil {
				slog.Info("cache hit", "type", patternType, "seed", musicCtx.VariationSeed)
				return &spec, nil
			}
		}
	}

	chordStr := progressionStringDetailed(musicCtx.ChordProgression)
	schemaExample := exampleSpecJSON(patternType)

	// REFACTOR.md point 1: Generate proper schema using invopop/jsonschema instead of manual inference
	schemaProps, schemaRequired, err := schema.GenerateJSONSchema()
	if err != nil {
		slog.Warn("failed to generate json schema, falling back to example inference", "error", err)
	}

	prompt := strings.ReplaceAll(string(promptTemplate), "{{KEY}}", musicCtx.Key.Root)
	prompt = strings.ReplaceAll(prompt, "{{MODE}}", musicCtx.Key.Mode)
	prompt = strings.ReplaceAll(prompt, "{{SCALE}}", musicCtx.Key.Scale)
	prompt = strings.ReplaceAll(prompt, "{{MODE_CHARACTER}}", modeCharacterDescription(musicCtx.Key))
	prompt = strings.ReplaceAll(prompt, "{{SCALE_NOTES}}", scaleNoteStringForPrompt(musicCtx.Key))
	prompt = strings.ReplaceAll(prompt, "{{BPM}}", fmt.Sprintf("%.0f", musicCtx.BPM))
	prompt = strings.ReplaceAll(prompt, "{{SEED}}", musicCtx.VariationSeed)
	prompt = strings.ReplaceAll(prompt, "{{CHORD_PROGRESSION}}", chordStr)
	prompt = strings.ReplaceAll(prompt, "{{MUSICAL_PLAN}}", plan.PromptText())
	prompt = strings.ReplaceAll(prompt, "{{SCHEMA}}", schemaExample)
	if !strings.Contains(string(promptTemplate), "{{MUSICAL_PLAN}}") {
		prompt += "\n\n**Musical plan:**\n" + plan.PromptText()
	}

	slog.Info("musical plan",
		"type", patternType,
		"seed", musicCtx.VariationSeed,
		"style_card", plan.StyleCard.Name,
		"motif", plan.MotifConcept,
		"tension_curve", plan.TensionCurve,
	)

	req := llm.GenerateRequest{
		System:           systemPrompt,
		Messages:         []llm.Message{{Role: "user", Content: prompt}},
		OutputSchema:     []byte(schemaExample),
		SchemaProperties: schemaProps,    // REFACTOR.md point 1: Pass pre-generated schema
		SchemaRequired:   schemaRequired, // REFACTOR.md point 1: Pass required fields
		Temperature:      0.3,
		MaxTokens:        4096,
	}

	validate := func(raw []byte) error {
		var spec schema.PatternSpec
		if err := json.Unmarshal(raw, &spec); err != nil {
			return fmt.Errorf("json parse: %w", err)
		}
		// LLMs sometimes write "minor_natural" in theory.scale even for modal keys (e.g. dorian).
		// Always use the actual key from the music context so note validation is correct.
		spec.Theory.Key = musicCtx.Key.Root
		spec.Theory.Scale = musicCtx.Key.Scale
		return g.validator.ValidateWithChords(&spec, musicCtx.ChordProgression)
	}

	slog.Debug("starting llm generation",
		"type", patternType,
		"provider", g.provider.Name(),
		"key", musicCtx.Key.Root,
		"scale", musicCtx.Key.Scale,
		"bpm", musicCtx.BPM,
		"bars", musicCtx.Bars,
		"seed", musicCtx.VariationSeed,
	)
	rawJSON, err := llm.GenerateWithRetry(ctx, g.provider, req, validate)
	if err != nil {
		var retryErr *llm.RetryExhaustedError
		if errors.As(err, &retryErr) && len(retryErr.LastRaw) > 0 {
			slog.Info("attempting programmatic repair",
				"type", patternType,
				"last_error", retryErr.Cause,
				"raw_bytes", len(retryErr.LastRaw),
				"raw_snippet", string(retryErr.LastRaw[:min(300, len(retryErr.LastRaw))]),
			)
			if repaired, repairErr := repairSpec(retryErr.LastRaw, musicCtx, validate); repairErr == nil {
				slog.Info("programmatic repair succeeded", "type", patternType)
				rawJSON = repaired
				err = nil
			} else {
				slog.Warn("programmatic repair failed",
					"type", patternType,
					"repair_err", repairErr,
					"original_err", retryErr.Cause,
				)
				return nil, err
			}
		} else {
			slog.Warn("llm generation failed, no raw output to repair",
				"type", patternType,
				"error", err,
			)
			return nil, err
		}
	}

	var spec schema.PatternSpec
	if err := json.Unmarshal(rawJSON, &spec); err != nil {
		return nil, fmt.Errorf("final parse: %w", err)
	}
	spec.Theory.Key = musicCtx.Key.Root
	spec.Theory.Scale = musicCtx.Key.Scale
	specPtr, finalJSON := g.critiqueAndMaybeRevise(ctx, req, rawJSON, &spec, musicCtx, plan)
	specPtr.Theory.Key = musicCtx.Key.Root
	specPtr.Theory.Scale = musicCtx.Key.Scale
	score := g.validator.ScoreMusicality(specPtr, musicCtx.ChordProgression)
	slog.Info("musical score",
		"type", patternType,
		"seed", musicCtx.VariationSeed,
		"repeated_phrases", score.RepeatedPhraseCount,
		"downbeat_chord_tone_ratio", score.DownbeatChordToneRatio,
		"pitch_contour_diversity", score.PitchContourDiversity,
		"velocity_flatness", score.VelocityFlatness,
		"warnings", score.Warnings,
	)
	if g.cache != nil {
		if err := g.cache.Set(finalJSON, cacheKeys...); err != nil {
			slog.Warn("cache write failed", "error", err)
		}
	}
	return specPtr, nil
}

func (g *SingleGenerator) loadOrBuildMusicalPlan(musicCtx MusicContext, patternType string) MusicalPlan {
	planKeys := []string{
		"musical-plan",
		patternType,
		musicCtx.Key.Root,
		musicCtx.Key.Mode,
		musicCtx.VariationSeed,
		plannerVersion,
		styleCardVersion,
		musicCtx.OfflineStyle,
	}
	if g.cache != nil {
		if cached, ok := g.cache.Get(planKeys...); ok {
			var plan MusicalPlan
			if err := json.Unmarshal(cached, &plan); err == nil && plan.PatternType != "" {
				return plan
			}
		}
	}

	plan := BuildMusicalPlan(musicCtx, patternType)
	if g.cache != nil {
		if raw, err := json.Marshal(plan); err == nil {
			if err := g.cache.Set(raw, planKeys...); err != nil {
				slog.Warn("plan cache write failed", "error", err)
			}
		}
	}
	return plan
}

func exampleSpecJSON(patternType string) string {
	var (
		octaveRange  [2]int
		steps        []schema.StepSpec
		styleProfile string
		automation   schema.AutomationIntent
	)

	switch patternType {
	case "melody":
		octaveRange = [2]int{4, 6}
		styleProfile = "melody_expressive"
		automation = schema.AutomationIntent{ModWheel: &schema.ModWheelIntent{Style: "moderate"}}
		steps = melodyExampleSteps()
	case "arpeggio":
		octaveRange = [2]int{3, 6}
		styleProfile = "arp_flowing"
		automation = schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: "subtle"}}
		steps = arpeggioExampleSteps()
	default: // bassline
		octaveRange = [2]int{1, 3}
		styleProfile = "bass_progressive"
		automation = schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: "medium"}}
		steps = basslineExampleSteps()
	}

	example := schema.PatternSpec{
		SpecVersion:  "1.0",
		PatternType:  patternType,
		Meta:         schema.PatternMeta{Name: "example", BPM: 122, Key: "Am", Bars: 16, Description: "example pattern"},
		Theory:       schema.TheorySpec{Key: "A", Mode: "minor", Scale: "minor_natural", OctaveRange: octaveRange},
		StyleProfile: styleProfile,
		Motif:        schema.MotifSpec{Length: 16, Steps: steps},
		Evolution: []schema.EvolutionStep{
			{FromBar: 1, ToBar: 4, Action: "introduce", Intensity: 0.3},
			{FromBar: 5, ToBar: 8, Action: "build", Intensity: 0.6},
			{FromBar: 9, ToBar: 12, Action: "peak", Intensity: 0.9},
			{FromBar: 13, ToBar: 16, Action: "release", Intensity: 0.5},
		},
		Automation:    automation,
		VariationSeed: "example-seed",
	}
	b, _ := json.MarshalIndent(example, "", "  ")
	return string(b)
}

// basslineExampleSteps returns a valid 16-step bassline motif (11 active, range A1-G3).
func basslineExampleSteps() []schema.StepSpec {
	return []schema.StepSpec{
		{Active: true, Note: "A2", Accent: true},
		{Active: true, Note: "A2"},
		{Active: false},
		{Active: true, Note: "E2", Slide: true},
		{Active: true, Note: "A2", Accent: true},
		{Active: false},
		{Active: true, Note: "A2", Ghost: true},
		{Active: true, Note: "E2"},
		{Active: true, Note: "A2", Accent: true},
		{Active: true, Note: "C3"},
		{Active: false},
		{Active: true, Note: "E2", Slide: true},
		{Active: true, Note: "A2", Accent: true},
		{Active: false},
		{Active: false},
		{Active: true, Note: "E2"},
	}
}

// arpeggioExampleSteps returns a valid 16-step arpeggio motif (14 active, range C3-C6).
func arpeggioExampleSteps() []schema.StepSpec {
	return []schema.StepSpec{
		{Active: true, Note: "A3", Legato: true},
		{Active: true, Note: "E4"},
		{Active: true, Note: "C5", Accent: true},
		{Active: true, Note: "A4"},
		{Active: true, Note: "E4", Legato: true},
		{Active: true, Note: "A3"},
		{Active: false},
		{Active: true, Note: "E4"},
		{Active: true, Note: "A4", Legato: true},
		{Active: true, Note: "C5", Accent: true},
		{Active: true, Note: "E5"},
		{Active: true, Note: "A4"},
		{Active: true, Note: "C5", Legato: true},
		{Active: true, Note: "E4"},
		{Active: false},
		{Active: true, Note: "A4"},
	}
}

// melodyExampleSteps returns a valid 16-step melody motif (7 active, range C4-C7).
func melodyExampleSteps() []schema.StepSpec {
	return []schema.StepSpec{
		{Active: true, Note: "A4", Accent: true},
		{Active: false},
		{Active: true, Note: "E5"},
		{Active: false},
		{Active: true, Note: "C5", Legato: true},
		{Active: false},
		{Active: false},
		{Active: true, Note: "A4"},
		{Active: false},
		{Active: false},
		{Active: true, Note: "E5", Accent: true},
		{Active: false},
		{Active: true, Note: "A5"},
		{Active: false},
		{Active: true, Note: "E5", Legato: true},
		{Active: false},
	}
}

func progressionString(prog theory.ChordProgression) string {
	if len(prog.Chords) == 0 {
		return ""
	}
	parts := make([]string, len(prog.Chords))
	for i, c := range prog.Chords {
		parts[i] = fmt.Sprintf("%s%s", c.Root, theory.QualitySuffix(c.Quality))
	}
	return strings.Join(parts, " → ")
}

func progressionStringDetailed(prog theory.ChordProgression) string {
	if len(prog.Chords) == 0 {
		return ""
	}
	parts := make([]string, len(prog.Chords))
	for i, c := range prog.Chords {
		notes, _ := theory.ChordNotes(c.Root, c.Quality)
		notesStr := strings.Join(notes, ", ")
		parts[i] = fmt.Sprintf("bars %d-%d: %s%s (notes: %s)",
			c.Bars[0], c.Bars[1], c.Root, theory.QualitySuffix(c.Quality), notesStr)
	}
	return strings.Join(parts, "\n")
}

// hashContent computes SHA-256 hash of content for cache key generation.
// REFACTOR.md point 6: Used to invalidate cache when prompt templates change.
func hashContent(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:8]) // Use first 8 bytes for brevity
}

func modeCharacterDescription(key theory.Key) string {
	scale := scaleNoteStringForPrompt(key)
	switch key.Scale {
	case "major":
		return fmt.Sprintf("Ionian / major (%s): bright and anthemic; character intervals are major 3rd, major 6th, and major 7th.", scale)
	case "minor_natural":
		return fmt.Sprintf("Aeolian / natural minor (%s): dark, introspective, club-friendly; character intervals are minor 3rd, minor 6th, and minor 7th.", scale)
	case "minor_harmonic":
		return fmt.Sprintf("Harmonic minor (%s): dark with a leading-tone pull; character intervals are minor 3rd, minor 6th, and major 7th.", scale)
	case "dorian":
		return fmt.Sprintf("Dorian (%s): dark groove with a hopeful lift; character intervals are minor 3rd, raised 6th, and minor 7th.", scale)
	case "phrygian":
		return fmt.Sprintf("Phrygian (%s): tense and shadowy; character intervals are flat 2nd, minor 3rd, and minor 7th.", scale)
	case "mixolydian":
		return fmt.Sprintf("Mixolydian (%s): open and festival-ready; character intervals are major 3rd and flat 7th.", scale)
	case "lydian":
		return fmt.Sprintf("Lydian (%s): floating and luminous; character intervals are major 3rd, raised 4th, and major 7th.", scale)
	default:
		return fmt.Sprintf("%s (%s): use chord tones first and treat scale color notes as tension.", key.Mode, scale)
	}
}

func scaleNoteStringForPrompt(key theory.Key) string {
	notes, err := theory.ScaleNotes(key.Root, key.Scale)
	if err != nil || len(notes) == 0 {
		return key.Root
	}
	return strings.Join(notes, " ")
}
