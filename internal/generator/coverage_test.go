package generator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/cache"
	"github.com/Andrea-Cavallo/cadenza/internal/llm"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

func TestDrumPattern_BasicStructure(t *testing.T) {
	events := GenerateDrumPattern(122, 16, "seed-a")
	if len(events) == 0 {
		t.Fatal("expected drum events")
	}

	var kickCount, clapCount, hatCount int
	for _, ev := range events {
		if ev.Channel != drumChannel {
			t.Fatalf("unexpected channel %d", ev.Channel)
		}
		if ev.Velocity > 120 {
			t.Fatalf("velocity too high: %d", ev.Velocity)
		}
		switch ev.Note {
		case kickNote:
			kickCount++
		case clapNote:
			clapCount++
		case closedHHNote:
			hatCount++
		}
	}

	if kickCount == 0 || clapCount == 0 || hatCount == 0 {
		t.Fatalf("expected kick/clap/hat coverage, got kick=%d clap=%d hat=%d", kickCount, clapCount, hatCount)
	}
}

func TestDrumPattern_LongFormAddsVariation(t *testing.T) {
	shortEvents := GenerateDrumPattern(122, 16, "seed-b")
	longEvents := GenerateDrumPattern(122, 32, "seed-b")

	var shortOpen, longOpen int
	for _, ev := range shortEvents {
		if ev.Note == openHHNote {
			shortOpen++
		}
	}
	for _, ev := range longEvents {
		if ev.Note == openHHNote {
			longOpen++
		}
	}

	if longOpen <= shortOpen {
		t.Fatalf("expected more open hats in long form: short=%d long=%d", shortOpen, longOpen)
	}
}

func TestGeneratorHelpers_ReturnUsefulData(t *testing.T) {
	prog := theory.ChordProgression{
		Key:  "A",
		Mode: "minor",
		Chords: []theory.ProgressionChord{
			{Root: "A", Quality: "minor", Bars: [2]int{1, 4}},
			{Root: "F", Quality: "major", Bars: [2]int{5, 8}},
		},
	}

	if got := progressionStringDetailed(prog); !strings.Contains(got, "bars 1-4") || !strings.Contains(got, "Am") {
		t.Fatalf("unexpected progression string: %q", got)
	}

	if hashA, hashB := hashContent([]byte("alpha")), hashContent([]byte("alpha")); hashA != hashB || len(hashA) == 0 {
		t.Fatalf("hashContent should be deterministic: %q %q", hashA, hashB)
	}
	if got := modeCharacterDescription(theory.Key{Root: "D", Mode: "dorian", Scale: "dorian"}); !strings.Contains(got, "raised 6th") {
		t.Fatalf("expected Dorian mode character to mention raised 6th, got %q", got)
	}

	example := exampleSpecJSON("bassline")
	var spec schema.PatternSpec
	if err := json.Unmarshal([]byte(example), &spec); err != nil {
		t.Fatalf("exampleSpecJSON should produce valid JSON: %v", err)
	}
	if spec.PatternType != "bassline" {
		t.Fatalf("unexpected pattern type %q", spec.PatternType)
	}
}

func TestMultiGenerator_AdditionalBranches(t *testing.T) {
	ctx := MusicContext{
		BPM: 122,
		Key: theory.Key{Root: "A", Mode: "minor", Scale: "minor_natural"},
		ChordProgression: theory.ChordProgression{
			Key: "A", Mode: "minor",
			Chords: []theory.ProgressionChord{
				{Root: "A", Quality: "minor", Bars: [2]int{1, 4}},
				{Root: "F", Quality: "major", Bars: [2]int{5, 8}},
				{Root: "C", Quality: "major", Bars: [2]int{9, 12}},
				{Root: "G", Quality: "major", Bars: [2]int{13, 16}},
			},
		},
		VariationSeed: "seed-c",
		Groove:        "straight",
		Bars:          16,
	}

	mg := newTestMultiGenerator(t, t.TempDir())
	if _, err := mg.GenerateWithContext(context.Background(), ctx, 16, 2); err != nil {
		t.Fatalf("GenerateWithContext error: %v", err)
	}

	spec, err := mg.generatePattern(context.Background(), ctx, "unknown")
	if err == nil || spec != nil {
		t.Fatal("expected unknown offline template to fail")
	}

	results, err := GenerateAll(context.Background(), Config{
		BPM:       122,
		Key:       theory.Key{Root: "A", Mode: "minor", Scale: "minor_natural"},
		Provider:  "claude",
		Model:     "unused",
		NoLLM:     true,
		OutputDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("GenerateAll error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 pattern results, got %d", len(results))
	}

	if _, err := buildProvider(Config{Provider: "nope"}); err == nil {
		t.Fatal("expected unsupported provider error")
	}
	if modeFlag(theory.Key{Mode: "minor"}) != "m" || modeFlag(theory.Key{Mode: "major"}) != "" {
		t.Fatal("modeFlag mismatch")
	}
}

func TestOfflineProfileHelpers_Branches(t *testing.T) {
	minorKey := theory.Key{Root: "A", Mode: "minor", Scale: "minor_natural"}
	majorKey := theory.Key{Root: "C", Mode: "major", Scale: "major"}

	hash := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	if chooseBassProfile(minorKey, 110, hash) == "" {
		t.Fatal("chooseBassProfile should return profile")
	}
	if chooseArpProfile(majorKey, 130, hash) == "" {
		t.Fatal("chooseArpProfile should return profile")
	}
	if chooseMelodyProfile(minorKey, 122, hash) == "" {
		t.Fatal("chooseMelodyProfile should return profile")
	}

	if buildBassEvolution(122, minorKey) == nil || buildArpEvolution(122, majorKey) == nil || buildMelodyEvolution(122, minorKey) == nil {
		t.Fatal("expected evolution helpers to return data")
	}
}

func TestPublicOfflineTemplateAndProviderNames(t *testing.T) {
	ctx := MusicContext{
		BPM:              122,
		Key:              theory.Key{Root: "A", Mode: "minor", Scale: "minor_natural"},
		ChordProgression: theory.SelectProgression("A", "minor_natural", "seed-d"),
		VariationSeed:    "seed-d",
		Bars:             16,
	}

	if OfflineTemplate("melody", ctx) == nil {
		t.Fatal("expected public offline template to return melody spec")
	}
	mock, err := buildProvider(Config{NoLLM: true})
	if err != nil {
		t.Fatalf("NoLLM buildProvider error: %v", err)
	}
	if mock.Name() != (&llm.MockProvider{}).Name() {
		t.Fatalf("unexpected provider name %q", mock.Name())
	}
}

func TestSingleGenerator_GenerateBranches(t *testing.T) {
	ctx := MusicContext{
		BPM:              122,
		Key:              theory.Key{Root: "A", Mode: "minor", Scale: "minor_natural"},
		ChordProgression: theory.SelectProgression("A", "minor_natural", "seed-e"),
		VariationSeed:    "seed-e",
		Bars:             16,
	}
	testSingleGeneratorMissingPrompt(t, ctx)
	testSingleGeneratorCacheHit(t, ctx)
	testSingleGeneratorInvalidCacheFallback(t, ctx)
}

func TestOfflineHelpers_AdditionalBranches(t *testing.T) {
	minorKey := theory.Key{Root: "A", Mode: "minor", Scale: "minor_natural"}
	majorKey := theory.Key{Root: "C", Mode: "major", Scale: "major"}
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i)
	}

	assertOfflineStringHelpers(t, minorKey)
	assertOfflineChordHelpers(t, minorKey)
	assertOfflineProfileSelection(t, minorKey, majorKey, hash)
	assertOfflineSequenceBuilders(t, minorKey, hash)
}

func TestDrumHelpers_DirectBranches(t *testing.T) {
	if events := appendKickEvents(nil, 0, 0, 0, 16, 0); len(events) != 1 || events[0].Velocity != 110 {
		t.Fatalf("expected accented downbeat kick, got %#v", events)
	}
	if events := appendKickEvents(nil, 120, 1, 16, 32, 0); len(events) != 1 || events[0].Velocity != 85 {
		t.Fatalf("expected long-form fill kick, got %#v", events)
	}
	if events := appendClapEvents(nil, 0, 4, 48, 16); len(events) != 1 || events[0].Velocity != 105 {
		t.Fatalf("expected boosted clap velocity, got %#v", events)
	}
	if events := appendClosedHHEvents(nil, 0, 2); len(events) != 1 || events[0].Velocity != 60 {
		t.Fatalf("expected non-downbeat hat velocity 60, got %#v", events)
	}
	if events := appendOpenHHEvents(nil, 0, 6, 118, 32, 16); len(events) != 1 || events[0].Velocity != 75 {
		t.Fatalf("expected open hat branch, got %#v", events)
	}
}

const generatorPromptTemplate = "{{KEY}} {{MODE}} {{SCALE}} {{BPM}} {{SEED}} {{CHORD_PROGRESSION}} {{SCHEMA}}"

func testSingleGeneratorMissingPrompt(t *testing.T, ctx MusicContext) {
	t.Helper()
	t.Run("missing prompt", func(t *testing.T) {
		g := NewGenerator(&llm.MockProvider{}, schema.NewValidator(), nil)
		g.promptDir = filepath.Join(t.TempDir(), "missing")
		if _, err := g.Generate(context.Background(), ctx, "bassline"); err == nil || !strings.Contains(err.Error(), "read prompt") {
			t.Fatalf("expected prompt read error, got %v", err)
		}
	})
}

func testSingleGeneratorCacheHit(t *testing.T, ctx MusicContext) {
	t.Helper()
	t.Run("cache hit bypasses provider", func(t *testing.T) {
		tmp := t.TempDir()
		c := cache.New(30, filepath.Join(tmp, "cache"))
		g := NewGenerator(&llm.MockProvider{Err: errors.New("provider should not be called")}, schema.NewValidator(), c)
		g.promptDir = filepath.Join(tmp, "prompts")
		writeGeneratorPrompt(t, g.promptDir, generatorPromptTemplate)

		cachedSpec := offlineTemplate("bassline", ctx)
		raw, err := json.Marshal(cachedSpec)
		if err != nil {
			t.Fatalf("marshal cached spec: %v", err)
		}
		if err := c.Set(raw, generatorCacheKeys(g, ctx, generatorPromptTemplate)...); err != nil {
			t.Fatalf("seed cache: %v", err)
		}

		spec, err := g.Generate(context.Background(), ctx, "bassline")
		if err != nil {
			t.Fatalf("unexpected generate error: %v", err)
		}
		if spec.PatternType != "bassline" {
			t.Fatalf("unexpected pattern type %q", spec.PatternType)
		}
	})
}

func testSingleGeneratorInvalidCacheFallback(t *testing.T, ctx MusicContext) {
	t.Helper()
	t.Run("invalid cached entry falls back to provider", func(t *testing.T) {
		tmp := t.TempDir()
		c := cache.New(30, filepath.Join(tmp, "cache"))
		validSpec := offlineTemplate("bassline", ctx)
		raw, err := json.Marshal(validSpec)
		if err != nil {
			t.Fatalf("marshal spec: %v", err)
		}
		g := NewGenerator(&llm.MockProvider{Response: raw}, schema.NewValidator(), c)
		g.promptDir = filepath.Join(tmp, "prompts")
		writeGeneratorPrompt(t, g.promptDir, generatorPromptTemplate)

		if err := c.Set([]byte("not-json"), generatorCacheKeys(g, ctx, generatorPromptTemplate)...); err != nil {
			t.Fatalf("seed bad cache entry: %v", err)
		}

		spec, err := g.Generate(context.Background(), ctx, "bassline")
		if err != nil {
			t.Fatalf("unexpected generate error: %v", err)
		}
		if spec.VariationSeed != ctx.VariationSeed {
			t.Fatalf("expected provider fallback result, got seed %q", spec.VariationSeed)
		}
	})
}

func writeGeneratorPrompt(t *testing.T, promptDir, prompt string) {
	t.Helper()
	if err := os.MkdirAll(promptDir, 0o755); err != nil {
		t.Fatalf("mkdir prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptDir, "bassline_v1.md"), []byte(prompt), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
}

func generatorCacheKeys(g *SingleGenerator, ctx MusicContext, prompt string) []string {
	return []string{g.provider.Name(), "bassline", ctx.Key.Root, ctx.Key.Mode, ctx.VariationSeed, hashContent([]byte(prompt))}
}

func assertOfflineStringHelpers(t *testing.T, minorKey theory.Key) {
	t.Helper()
	if got := progressionString(theory.ChordProgression{}); got != "" {
		t.Fatalf("expected empty progression string, got %q", got)
	}
	if got := progressionStringDetailed(theory.ChordProgression{}); got != "" {
		t.Fatalf("expected empty detailed progression string, got %q", got)
	}
	if got := approachNote("A", minorKey); got == "" || got == "A3" {
		t.Fatalf("unexpected approach note %q", got)
	}
	if got := approachNote("?", minorKey); got != "?" {
		t.Fatalf("expected invalid root fallback, got %q", got)
	}
}

func assertOfflineChordHelpers(t *testing.T, minorKey theory.Key) {
	t.Helper()
	if got := chordFifth(theory.ProgressionChord{Root: "?", Quality: "bad"}); got != "?" {
		t.Fatalf("expected root fallback, got %q", got)
	}
	if got := closestChordTone([]string{"?"}, []string{"A"}, 0); got != "?" {
		t.Fatalf("expected first chord tone fallback, got %q", got)
	}
}

func assertOfflineProfileSelection(t *testing.T, minorKey, majorKey theory.Key, hash []byte) {
	t.Helper()
	if chooseBassProfile(majorKey, 110, hash) == "" {
		t.Fatal("expected low-bpm bass profile")
	}
	if prof := chooseBassProfile(minorKey, 130, hash); prof != "bass_driving" {
		t.Fatalf("expected minor high-bpm driving bass, got %q", prof)
	}
	if prof := chooseArpProfile(majorKey, 110, hash); prof != "arp_flowing" {
		t.Fatalf("expected flowing arp, got %q", prof)
	}
	if chooseArpProfile(minorKey, 125, hash) == "" {
		t.Fatal("expected minor mid-bpm arp profile")
	}
	if prof := chooseMelodyProfile(majorKey, 110, hash); prof != "melody_expressive" {
		t.Fatalf("expected expressive melody, got %q", prof)
	}
	if chooseMelodyProfile(majorKey, 126, hash) == "" {
		t.Fatal("expected major fast melody profile")
	}
}

func assertOfflineSequenceBuilders(t *testing.T, minorKey theory.Key, hash []byte) {
	t.Helper()
	if len(buildBassEvolution(129, minorKey)) != 4 {
		t.Fatal("expected 4 bass evolution phases")
	}
	if len(buildHypnoticMotif(hash, 5, []string{"A", "B", "C", "D", "E", "F", "G"})) != 5 {
		t.Fatal("expected motif length 5")
	}
	if got := bassColorTone(minorKey, theory.ProgressionChord{Root: "F", Quality: "major"}); got != "F" {
		t.Fatalf("expected A minor color tone F over F major, got %q", got)
	}
	lydianKey := theory.Key{Root: "C", Mode: "lydian", Scale: "lydian"}
	if got := bassColorTone(lydianKey, theory.ProgressionChord{Root: "D", Quality: "major"}); got != "F#" {
		t.Fatalf("expected C lydian raised fourth F# over D major, got %q", got)
	}
	if got := scaleDegreeName(lydianKey, -1); got != "B" {
		t.Fatalf("expected wrapped scale degree B, got %q", got)
	}
	if chooseArpPattern(hash, minorKey, 0) == chooseArpPattern(hash, lydianKey, 0) {
		t.Fatal("expected key character to bias arpeggio pattern selection")
	}
	if len(seedHash("abc")) != 32 {
		t.Fatal("expected sha256 hash bytes")
	}
}

// TestOfflineSeedDiversity asserts that seeds 1-20 produce meaningfully different
// rhythmic patterns for all three pattern types. "Different" means distinct step
// activity bitmasks — not just different root notes.
func TestOfflineSeedDiversity(t *testing.T) {
	amKey := theory.Key{Root: "A", Scale: "minor_natural", Mode: "minor"}
	prog := theory.SelectProgression("A", "minor_natural", "base-seed")

	baseCtx := MusicContext{Key: amKey, ChordProgression: prog, BPM: 124}

	patternTypes := []string{"bassline", "arpeggio", "melody"}
	for _, pt := range patternTypes {
		t.Run(pt, func(t *testing.T) {
			fingerprints := make(map[string]bool)
			for seed := 1; seed <= 20; seed++ {
				ctx := baseCtx
				ctx.VariationSeed = fmt.Sprintf("seed-%d", seed)
				spec := offlineTemplate(pt, ctx)
				if spec == nil {
					t.Fatalf("seed %d: nil spec", seed)
				}
				fingerprints[stepActivityFingerprint(spec.Motif.Steps)] = true
			}
			if len(fingerprints) < 6 {
				t.Errorf("%s: only %d distinct step patterns from 20 seeds (want ≥6)", pt, len(fingerprints))
			}
		})
	}
}

// TestOfflineKeyDifferentiation asserts that Am and Dm with the same seed
// produce different rhythmic patterns (not just different root notes).
func TestOfflineKeyDifferentiation(t *testing.T) {
	amKey := theory.Key{Root: "A", Scale: "minor_natural", Mode: "minor"}
	dmKey := theory.Key{Root: "D", Scale: "minor_natural", Mode: "minor"}

	amProg := theory.SelectProgression("A", "minor_natural", "seed-X")
	dmProg := theory.SelectProgression("D", "minor_natural", "seed-X")

	for _, pt := range []string{"bassline", "arpeggio", "melody"} {
		t.Run(pt, func(t *testing.T) {
			amSpec := offlineTemplate(pt, MusicContext{Key: amKey, ChordProgression: amProg, BPM: 124, VariationSeed: "seed-X"})
			dmSpec := offlineTemplate(pt, MusicContext{Key: dmKey, ChordProgression: dmProg, BPM: 124, VariationSeed: "seed-X"})

			if amSpec == nil || dmSpec == nil {
				t.Fatal("nil spec")
			}
			amFP := stepActivityFingerprint(amSpec.Motif.Steps)
			dmFP := stepActivityFingerprint(dmSpec.Motif.Steps)
			if amFP == dmFP {
				t.Errorf("%s: Am and Dm produce identical step patterns with same seed", pt)
			}
		})
	}
}

// stepActivityFingerprint returns a compact string representing which steps are active,
// accented, ghost, or slide — capturing rhythmic structure independent of root note.
func TestOfflineSubModesValidateAndShapeDensity(t *testing.T) {
	key := theory.Key{Root: "A", Scale: "minor_natural", Mode: "minor"}
	prog := theory.SelectProgression("A", "minor_natural", "style-seed")
	validator := schema.NewValidator()

	for _, style := range []string{offlineStyleMelodic, offlineStyleHypnotic, offlineStyleDriving, offlineStyleMinimal} {
		t.Run(style, func(t *testing.T) {
			ctx := MusicContext{
				BPM:              124,
				Key:              key,
				ChordProgression: prog,
				VariationSeed:    "style-seed",
				OfflineStyle:     style,
			}
			for _, patternType := range []string{"bassline", "arpeggio", "melody"} {
				spec := offlineTemplate(patternType, ctx)
				if spec == nil {
					t.Fatalf("%s: nil spec", patternType)
				}
				if err := validator.ValidateWithChords(spec, prog); err != nil {
					t.Fatalf("%s %s did not validate: %v", style, patternType, err)
				}
				assertGateVariation(t, spec)
			}
		})
	}

	hypnoticBass := offlineTemplate("bassline", MusicContext{BPM: 124, Key: key, ChordProgression: prog, VariationSeed: "style-seed", OfflineStyle: offlineStyleHypnotic})
	if got := activeStepCount(hypnoticBass.Motif.Steps); got != 8 {
		t.Fatalf("hypnotic bass should stay sparse at 8 active steps, got %d", got)
	}
	drivingArp := offlineTemplate("arpeggio", MusicContext{BPM: 124, Key: key, ChordProgression: prog, VariationSeed: "style-seed", OfflineStyle: offlineStyleDriving})
	if got := activeStepCount(drivingArp.Motif.Steps); got != 16 {
		t.Fatalf("driving arpeggio should fill all 16 steps, got %d", got)
	}
	minimalMelody := offlineTemplate("melody", MusicContext{BPM: 124, Key: key, ChordProgression: prog, VariationSeed: "style-seed", OfflineStyle: offlineStyleMinimal})
	if got := activeStepCount(minimalMelody.Motif.Steps); got != 4 {
		t.Fatalf("minimal melody should use exactly 4 active steps, got %d", got)
	}
}

func TestOfflineRhythmicFigureCountAndPassingNotes(t *testing.T) {
	key := theory.Key{Root: "A", Scale: "minor_natural", Mode: "minor"}
	prog := theory.SelectProgression("A", "minor_natural", "figure-seed")
	baseCtx := MusicContext{BPM: 124, Key: key, ChordProgression: prog}

	for _, patternType := range []string{"bassline", "arpeggio", "melody"} {
		fingerprints := make(map[string]bool)
		for seed := 1; seed <= 20; seed++ {
			ctx := baseCtx
			ctx.VariationSeed = fmt.Sprintf("figure-%d", seed)
			spec := offlineTemplate(patternType, ctx)
			fingerprints[stepActivityFingerprint(spec.Motif.Steps)] = true
		}
		if len(fingerprints) < 4 {
			t.Fatalf("%s has only %d rhythmic figures, want at least 4", patternType, len(fingerprints))
		}
	}

	foundPassing := false
	for seed := 1; seed <= 100 && !foundPassing; seed++ {
		ctx := baseCtx
		ctx.VariationSeed = fmt.Sprintf("passing-%d", seed)
		spec := offlineTemplate("melody", ctx)
		for _, step := range spec.Motif.Steps {
			if step.Active && step.Ghost && step.Staccato {
				foundPassing = true
				break
			}
		}
	}
	if !foundPassing {
		t.Fatal("expected deterministic passing-note probability to create at least one ghost staccato passing note")
	}
}

func stepActivityFingerprint(steps []schema.StepSpec) string {
	b := make([]byte, len(steps))
	for i, s := range steps {
		var v byte
		if s.Active {
			v |= 1
		}
		if s.Accent {
			v |= 2
		}
		if s.Ghost {
			v |= 4
		}
		if s.Slide {
			v |= 8
		}
		b[i] = '0' + v
	}
	return string(b)
}

func assertGateVariation(t *testing.T, spec *schema.PatternSpec) {
	t.Helper()
	hasLegato, hasStaccato := false, false
	for _, step := range spec.Motif.Steps {
		hasLegato = hasLegato || step.Legato
		hasStaccato = hasStaccato || step.Staccato
	}
	if !hasLegato || !hasStaccato {
		t.Fatalf("%s should mix legato and staccato gates, legato=%v staccato=%v", spec.PatternType, hasLegato, hasStaccato)
	}
}
