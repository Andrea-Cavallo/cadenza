package generator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Andrea-Cavallo/cadenza/internal/cache"
	"github.com/Andrea-Cavallo/cadenza/internal/llm"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// MusicContext carries all musical parameters for a generation pass.
type MusicContext struct {
	BPM              float64
	Key              theory.Key
	Bars             int
	VariationSeed    string
	ChordProgression theory.ChordProgression
	Groove           string // timing preset: straight, mpc60, linndrum, humanize
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
		promptDir: "prompts",
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
- Return ONLY valid JSON, no markdown, no explanation`

func (g *SingleGenerator) Generate(ctx context.Context, musicCtx MusicContext, patternType string) (*schema.PatternSpec, error) {
	promptPath := filepath.Join(g.promptDir, patternType+"_v1.md")
	promptTemplate, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("read prompt %q: %w", promptPath, err)
	}

	// REFACTOR.md point 6: Include prompt template hash in cache key for invalidation on prompt changes
	promptHash := hashContent(promptTemplate)
	cacheKeys := []string{g.provider.Name(), patternType, musicCtx.Key.Root, musicCtx.Key.Mode, musicCtx.VariationSeed, promptHash}

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
	prompt = strings.ReplaceAll(prompt, "{{BPM}}", fmt.Sprintf("%.0f", musicCtx.BPM))
	prompt = strings.ReplaceAll(prompt, "{{SEED}}", musicCtx.VariationSeed)
	prompt = strings.ReplaceAll(prompt, "{{CHORD_PROGRESSION}}", chordStr)
	prompt = strings.ReplaceAll(prompt, "{{SCHEMA}}", schemaExample)

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
		return g.validator.ValidateWithChords(&spec, musicCtx.ChordProgression)
	}

	rawJSON, err := llm.GenerateWithRetry(ctx, g.provider, req, validate)
	if err != nil {
		return nil, err
	}

	if g.cache != nil {
		if err := g.cache.Set(rawJSON, cacheKeys...); err != nil {
			slog.Warn("cache write failed", "error", err)
		}
	}

	var spec schema.PatternSpec
	if err := json.Unmarshal(rawJSON, &spec); err != nil {
		return nil, fmt.Errorf("final parse: %w", err)
	}
	return &spec, nil
}

func exampleSpecJSON(patternType string) string {
	example := schema.PatternSpec{
		SpecVersion:  "1.0",
		PatternType:  patternType,
		Meta:         schema.PatternMeta{Name: "example", BPM: 122, Key: "Am", Bars: 16, Description: "example pattern"},
		Theory:       schema.TheorySpec{Key: "A", Mode: "minor", Scale: "minor_natural", OctaveRange: [2]int{2, 3}},
		StyleProfile: "bass_progressive",
		Motif: schema.MotifSpec{
			Length: 16,
			Steps: []schema.StepSpec{
				{Active: true, Note: "A2", Accent: true},
				{Active: true, Note: "A2"},
				{Active: false},
				{Active: true, Note: "E2", Slide: true},
			},
		},
		Evolution: []schema.EvolutionStep{
			{FromBar: 1, ToBar: 4, Action: "introduce", Intensity: 0.3},
		},
		Automation:    schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: "medium"}},
		VariationSeed: "example-seed",
	}
	b, _ := json.MarshalIndent(example, "", "  ")
	return string(b)
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
