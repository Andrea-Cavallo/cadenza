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
	midipkg "github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
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

func TestMultiGenerator_LLMFailureFallsBackOffline(t *testing.T) {
	ctx := MusicContext{
		BPM:              122,
		Key:              theory.Key{Root: "A", Mode: "minor", Scale: "minor_natural"},
		ChordProgression: theory.SelectProgression("A", "minor_natural", "fallback-seed"),
		VariationSeed:    "fallback-seed",
		Groove:           "straight",
		Bars:             16,
	}
	v := schema.NewValidator()
	reg := styleprofile.NewRegistry()
	rend := renderer.New()
	w := midipkg.NewWriter(122)
	mg := NewMultiGenerator(&llm.MockProvider{Err: errors.New("llm unavailable")}, v, reg, rend, w, t.TempDir())

	spec, err := mg.generatePattern(context.Background(), ctx, "melody")
	if err != nil {
		t.Fatalf("expected offline fallback after LLM failure, got %v", err)
	}
	if spec.PatternType != "melody" || spec.VariationSeed != ctx.VariationSeed {
		t.Fatalf("unexpected fallback spec %+v", spec)
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
	testSingleGeneratorInjectsMusicalPlan(t, ctx)
	testSingleGeneratorCriticRevision(t, ctx)
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

func TestBuildMusicalPlan_StyleCardsAndPromptText(t *testing.T) {
	ctx := MusicContext{
		BPM:              130,
		Key:              theory.Key{Root: "D", Mode: "dorian", Scale: "dorian"},
		ChordProgression: theory.SelectProgression("D", "dorian", "plan-seed"),
		VariationSeed:    "plan-seed",
		OfflineStyle:     "driving",
	}
	plan := BuildMusicalPlan(ctx, "arpeggio")
	if plan.StyleCard.Name != "warehouse-driving" {
		t.Fatalf("expected driving style card, got %q", plan.StyleCard.Name)
	}
	text := plan.PromptText()
	for _, want := range []string{"Style card:", "Section intentions:", "Revision priorities", "arpeggio direction"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected plan text to contain %q, got %q", want, text)
		}
	}

	ctx.OfflineStyle = ""
	ctx.BPM = 122
	plan = BuildMusicalPlan(ctx, "melody")
	if plan.StyleCard.Name != "afterlife-dark" {
		t.Fatalf("expected Dorian to map to afterlife-dark, got %q", plan.StyleCard.Name)
	}
	if !strings.Contains(plan.MotifConcept, "tension") {
		t.Fatalf("expected melody motif concept to mention tension, got %q", plan.MotifConcept)
	}
}

func TestGoldenPromptContracts(t *testing.T) {
	for _, promptName := range []string{"bassline_v1.md", "arpeggio_v1.md", "melody_v1.md"} {
		t.Run(promptName, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "prompts", promptName))
			if err != nil {
				t.Fatalf("read prompt: %v", err)
			}
			prompt := string(raw)
			for _, want := range []string{"{{MUSICAL_PLAN}}", "{{MODE_CHARACTER}}", "{{CHORD_PROGRESSION}}", "Return ONLY valid JSON"} {
				if !strings.Contains(prompt, want) {
					t.Fatalf("prompt %s missing contract marker %q", promptName, want)
				}
			}
		})
	}
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

const generatorPromptTemplate = "{{KEY}} {{MODE}} {{SCALE}} {{BPM}} {{SEED}} {{CHORD_PROGRESSION}} {{MUSICAL_PLAN}} {{SCHEMA}}"

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
	plan := BuildMusicalPlan(ctx, "bassline")
	return []string{
		g.provider.Name(),
		"bassline",
		ctx.Key.Root,
		ctx.Key.Mode,
		ctx.VariationSeed,
		hashContent([]byte(prompt)),
		plannerVersion,
		styleCardVersion,
		plan.StyleCard.Name,
		criticVersion,
		revisionPolicyVersion,
	}
}

func testSingleGeneratorInjectsMusicalPlan(t *testing.T, ctx MusicContext) {
	t.Helper()
	t.Run("injects musical plan into prompt", func(t *testing.T) {
		tmp := t.TempDir()
		validSpec := offlineTemplate("bassline", ctx)
		raw, err := json.Marshal(validSpec)
		if err != nil {
			t.Fatalf("marshal spec: %v", err)
		}
		provider := &recordingProvider{response: raw}
		g := NewGenerator(provider, schema.NewValidator(), nil)
		g.promptDir = filepath.Join(tmp, "prompts")
		writeGeneratorPrompt(t, g.promptDir, generatorPromptTemplate)

		if _, err := g.Generate(context.Background(), ctx, "bassline"); err != nil {
			t.Fatalf("unexpected generate error: %v", err)
		}
		prompt := provider.lastRequest.Messages[0].Content
		for _, want := range []string{"Style card:", "Motif concept:", "Revision priorities", "prydz-progressive"} {
			if !strings.Contains(prompt, want) {
				t.Fatalf("expected prompt to contain %q, got %q", want, prompt)
			}
		}
	})
}

type recordingProvider struct {
	response    []byte
	lastRequest llm.GenerateRequest
}

func (p *recordingProvider) Name() string {
	return "recording"
}

func (p *recordingProvider) Generate(_ context.Context, req llm.GenerateRequest) (llm.GenerateResponse, error) {
	p.lastRequest = req
	return llm.GenerateResponse{RawJSON: p.response, Provider: p.Name()}, nil
}

func testSingleGeneratorCriticRevision(t *testing.T, ctx MusicContext) {
	t.Helper()
	t.Run("critic applies one targeted revision", func(t *testing.T) {
		tmp := t.TempDir()
		initial := offlineTemplate("bassline", ctx)
		revised := offlineTemplate("bassline", ctx)
		revised.Meta.Description = "revised by critic"

		initialRaw, err := json.Marshal(initial)
		if err != nil {
			t.Fatalf("marshal initial: %v", err)
		}
		revisedRaw, err := json.Marshal(revised)
		if err != nil {
			t.Fatalf("marshal revised: %v", err)
		}
		criticRaw := []byte(mustJSON(CriticReport{
			PatternType:             "bassline",
			NeedsRevision:           true,
			RepetitionScore:         0.4,
			MotifClarityScore:       0.8,
			ChordToneStrengthScore:  0.9,
			ContourScore:            0.7,
			DensityScore:            0.8,
			TensionArcScore:         0.5,
			TrackSeparationScore:    0.9,
			Problems:                []string{"section tension arc does not clearly build and resolve"},
			RevisionInstructions:    []string{"make bars 9-12 the tension point"},
			KeepElements:            []string{"keep downbeat chord tones"},
			RevisionPromptVersion:   revisionPolicyVersion,
			OneRevisionRoundApplied: false,
		}))

		provider := &sequenceProvider{responses: [][]byte{initialRaw, criticRaw, revisedRaw}}
		g := NewGenerator(provider, schema.NewValidator(), nil)
		g.promptDir = filepath.Join(tmp, "prompts")
		writeGeneratorPrompt(t, g.promptDir, generatorPromptTemplate)

		spec, err := g.Generate(context.Background(), ctx, "bassline")
		if err != nil {
			t.Fatalf("unexpected generate error: %v", err)
		}
		if spec.Meta.Description != "revised by critic" {
			t.Fatalf("expected revised spec, got description %q", spec.Meta.Description)
		}
		if provider.calls != 3 {
			t.Fatalf("expected initial + critic + one revision call, got %d", provider.calls)
		}
		if !strings.Contains(provider.requests[2].Messages[len(provider.requests[2].Messages)-1].Content, "Revise the previous PatternSpec once") {
			t.Fatalf("expected targeted revision prompt, got %#v", provider.requests[2].Messages)
		}
	})
}

type sequenceProvider struct {
	responses [][]byte
	requests  []llm.GenerateRequest
	calls     int
}

func (p *sequenceProvider) Name() string {
	return "sequence"
}

func (p *sequenceProvider) Generate(_ context.Context, req llm.GenerateRequest) (llm.GenerateResponse, error) {
	p.requests = append(p.requests, req)
	idx := p.calls
	p.calls++
	if idx >= len(p.responses) {
		idx = len(p.responses) - 1
	}
	return llm.GenerateResponse{RawJSON: p.responses[idx], Provider: p.Name()}, nil
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

	// 64-step motif (4 chord sections × 16 steps): expected counts scale ×4 from the 16-step era.
	hypnoticBass := offlineTemplate("bassline", MusicContext{BPM: 124, Key: key, ChordProgression: prog, VariationSeed: "style-seed", OfflineStyle: offlineStyleHypnotic})
	if got := activeStepCount(hypnoticBass.Motif.Steps); got != 32 {
		t.Fatalf("hypnotic bass should stay sparse at 32 active steps (2/sub-group×16), got %d", got)
	}
	drivingArp := offlineTemplate("arpeggio", MusicContext{BPM: 124, Key: key, ChordProgression: prog, VariationSeed: "style-seed", OfflineStyle: offlineStyleDriving})
	if got := activeStepCount(drivingArp.Motif.Steps); got != 64 {
		t.Fatalf("driving arpeggio should fill all 64 steps, got %d", got)
	}
	minimalMelody := offlineTemplate("melody", MusicContext{BPM: 124, Key: key, ChordProgression: prog, VariationSeed: "style-seed", OfflineStyle: offlineStyleMinimal})
	if got := activeStepCount(minimalMelody.Motif.Steps); got != 16 {
		t.Fatalf("minimal melody should use exactly 16 active steps (4 downbeats×4 sections), got %d", got)
	}
}

func TestOfflineStyleListeningFixture(t *testing.T) {
	type offlineStyleCase struct {
		Name         string   `json:"name"`
		BPM          float64  `json:"bpm"`
		Key          string   `json:"key"`
		Seed         string   `json:"seed"`
		Styles       []string `json:"styles"`
		PatternTypes []string `json:"patternTypes"`
	}

	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "listening", "offline_style_cases.json"))
	if err != nil {
		t.Fatalf("read offline style fixture: %v", err)
	}
	var cases []offlineStyleCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("decode offline style fixture: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("expected at least one offline style fixture case")
	}

	validator := schema.NewValidator()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			key, err := theory.ParseKey(tc.Key)
			if err != nil {
				t.Fatalf("parse key %q: %v", tc.Key, err)
			}
			prog := theory.SelectProgression(key.Root, key.Scale, tc.Seed)

			for _, patternType := range tc.PatternTypes {
				fingerprints := make(map[string]string)
				for _, style := range tc.Styles {
					ctx := MusicContext{
						BPM:              tc.BPM,
						Key:              key,
						ChordProgression: prog,
						VariationSeed:    tc.Seed,
						OfflineStyle:     style,
						Bars:             16,
					}
					spec := offlineTemplate(patternType, ctx)
					if spec == nil {
						t.Fatalf("%s %s: nil spec", patternType, style)
					}
					if err := validator.ValidateWithChords(spec, prog); err != nil {
						t.Fatalf("%s %s invalid: %v", patternType, style, err)
					}
					fp := musicalFingerprint(spec.Motif.Steps)
					if previousStyle, exists := fingerprints[fp]; exists {
						t.Fatalf("%s styles %q and %q produced the same musical fingerprint %q", patternType, previousStyle, style, fp)
					}
					fingerprints[fp] = style
				}
			}
		})
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

func TestBuildHypnoticMotifNoDoubleLeap(t *testing.T) {
	scaleNotes := []string{"A", "B", "C", "D", "E", "F", "G"}
	for seed := 1; seed <= 50; seed++ {
		h := seedHash(fmt.Sprintf("guard-seed-%d", seed))
		motif := buildHypnoticMotif(h, 5, scaleNotes)
		for i := 1; i < len(motif)-1; i++ {
			d1 := motif[i] - motif[i-1]
			d2 := motif[i+1] - motif[i]
			abs1 := d1
			if abs1 < 0 {
				abs1 = -abs1
			}
			abs2 := d2
			if abs2 < 0 {
				abs2 = -abs2
			}
			sameDir := (d1 > 0 && d2 > 0) || (d1 < 0 && d2 < 0)
			if sameDir && abs1 >= 3 && abs2 >= 3 {
				t.Errorf("seed %d: double same-direction leap at [%d,%d,%d]: intervals %d,%d",
					seed, i-1, i, i+1, d1, d2)
			}
		}
	}
}

func TestContourTemplatesHaveCorrectDensity(t *testing.T) {
	type densityCase struct {
		name                 ContourType
		rhythms              [3][stepsPerSection]byte
		minActive, maxActive int
	}
	cases := []densityCase{
		{ContourArch, archRhythms, 6, 8},
		{ContourQuestionAnswer, questionAnswerRhythms, 5, 8},
		{ContourTensionHold, tensionHoldRhythms, 7, 10},
		{ContourDescRelease, descReleaseRhythms, 4, 6},
	}
	for _, tc := range cases {
		for v, variant := range tc.rhythms {
			active := 0
			for _, code := range variant {
				if code > 0 {
					active++
				}
			}
			if active < tc.minActive || active > tc.maxActive {
				t.Errorf("%s variant %d: %d active steps, want [%d,%d]",
					tc.name, v, active, tc.minActive, tc.maxActive)
			}
		}
	}
}

func TestChooseSectionContourBias(t *testing.T) {
	for seed := 0; seed < 100; seed++ {
		h := seedHash(fmt.Sprintf("bias-%d", seed))
		ct := chooseSectionContour(2, h)
		if ct != ContourTensionHold && ct != ContourArch {
			t.Errorf("section 2 seed %d: got %q, want tension_hold or arch", seed, ct)
		}
		ct3 := chooseSectionContour(3, h)
		if ct3 != ContourDescRelease && ct3 != ContourQuestionAnswer {
			t.Errorf("section 3 seed %d: got %q, want descending_release or question_answer", seed, ct3)
		}
	}
}

func TestBuildMelodyContourStructure(t *testing.T) {
	key := theory.Key{Root: "A", Scale: "minor_natural", Mode: "minor"}
	prog := theory.SelectProgression("A", "minor_natural", "test-contour")
	scaleNotes, _ := theory.ScaleNotes("A", "minor_natural")
	h := seedHash("test-contour" + "A" + "minor_natural" + "melody")
	motif := buildHypnoticMotif(h, 4, scaleNotes)

	contour := buildMelodyContour(h, prog, motif, scaleNotes, key)

	if len(contour.Sections) != 4 {
		t.Fatalf("expected 4 sections, got %d", len(contour.Sections))
	}
	for i, intent := range contour.Sections[3].Intentions {
		if intent.Active && intent.PreferHigh {
			t.Errorf("section 3 step %d: PreferHigh should be false (resolution descends)", i)
		}
	}
	for sec := 0; sec < 4; sec++ {
		active := 0
		for _, intent := range contour.Sections[sec].Intentions {
			if intent.Active {
				active++
			}
		}
		if active < 4 {
			t.Errorf("section %d has only %d active steps (want >= 4)", sec, active)
		}
	}
}

func TestMelodyContourDiversity(t *testing.T) {
	key := theory.Key{Root: "A", Scale: "minor_natural", Mode: "minor"}
	prog := theory.SelectProgression("A", "minor_natural", "base-diversity")
	scaleNotes, _ := theory.ScaleNotes("A", "minor_natural")

	fingerprints := make(map[string]bool)
	for seed := 1; seed <= 10; seed++ {
		h := seedHash(fmt.Sprintf("diversity-seed-%d", seed) + "A" + "minor_natural" + "melody")
		motif := buildHypnoticMotif(h, 4, scaleNotes)
		contour := buildMelodyContour(h, prog, motif, scaleNotes, key)
		fp := fmt.Sprintf("%s|%s|%s|%s",
			contour.Sections[0].Type,
			contour.Sections[1].Type,
			contour.Sections[2].Type,
			contour.Sections[3].Type,
		)
		fingerprints[fp] = true
	}
	if len(fingerprints) < 3 {
		t.Errorf("expected >= 3 distinct contour combinations from 10 seeds, got %d", len(fingerprints))
	}
}

func TestWithinLeap(t *testing.T) {
	if !withinLeap("C", "D", 4) {
		t.Error("C→D should be within 4 semitones")
	}
	if !withinLeap("C", "G", 5) {
		t.Error("C→G (5 semitones wrap) should be within 5")
	}
	if withinLeap("C", "F#", 5) {
		t.Error("C→F# (6 semitones) should NOT be within 5")
	}
}

func TestResolveTargetNote_ReturnsValidNote(t *testing.T) {
	key := theory.Key{Root: "A", Scale: "minor_natural", Mode: "minor"}
	scaleNotes, _ := theory.ScaleNotes("A", "minor_natural")
	chordTones := []string{"A", "C", "E"}

	for degree := 0; degree < 7; degree++ {
		result := resolveTargetNote(chordTones, scaleNotes, degree, key)
		if result == "" {
			t.Errorf("degree %d: got empty note", degree)
		}
	}
}

func TestBuildMelodyFromContourBasicStructure(t *testing.T) {
	key := theory.Key{Root: "A", Scale: "minor_natural", Mode: "minor"}
	prog := theory.SelectProgression("A", "minor_natural", "fill-test")
	scaleNotes, _ := theory.ScaleNotes("A", "minor_natural")
	h := seedHash("fill-test" + "A" + "minor_natural" + "melody")
	motif := buildHypnoticMotif(h, 4, scaleNotes)
	contour := buildMelodyContour(h, prog, motif, scaleNotes, key)

	steps := make([]schema.StepSpec, motifSteps)
	buildMelodyFromContour(steps, contour, prog, scaleNotes, key, h)

	active := 0
	for _, s := range steps {
		if s.Active {
			active++
		}
	}
	if active == 0 {
		t.Fatal("expected active steps after fill pass")
	}

	for i, s := range steps {
		if !s.Active {
			continue
		}
		midi, err := theory.NoteToMIDI(s.Note)
		if err != nil {
			t.Errorf("step %d: invalid note %q: %v", i, s.Note, err)
			continue
		}
		if midi < 60 || midi > 84 {
			t.Errorf("step %d: note %q MIDI=%d outside [60,84]", i, s.Note, midi)
		}
	}

	for i := stepsPerSection*3 + 1; i < motifSteps; i++ {
		if steps[i].Active && steps[i].Accent {
			t.Errorf("resolution section step %d has unexpected Accent", i)
		}
	}
}

func TestMelodyNotePreferHigher(t *testing.T) {
	noteHigh := melodyNote("C", "5", true)
	midiHigh, err := theory.NoteToMIDI(noteHigh)
	if err != nil || midiHigh != 84 {
		t.Errorf("melodyNote(C,5,true): expected C6=84, got %s MIDI=%d err=%v", noteHigh, midiHigh, err)
	}

	noteLow := melodyNote("C", "5", false)
	midiLow, err := theory.NoteToMIDI(noteLow)
	if err != nil || midiLow != 72 {
		t.Errorf("melodyNote(C,5,false): expected C5=72, got %s MIDI=%d err=%v", noteLow, midiLow, err)
	}

	noteFallback := melodyNote("E", "5", true)
	midiFallback, err := theory.NoteToMIDI(noteFallback)
	if err != nil || midiFallback < 60 || midiFallback > 84 {
		t.Errorf("melodyNote(E,5,true): expected range [60,84], got %s MIDI=%d err=%v", noteFallback, midiFallback, err)
	}
}

func TestScoreMelodyPhrase_BasicMetrics(t *testing.T) {
	key := theory.Key{Root: "A", Scale: "minor_natural", Mode: "minor"}
	prog := theory.SelectProgression("A", "minor_natural", "scorer-seed")
	ctx := MusicContext{
		BPM:              122,
		Key:              key,
		ChordProgression: prog,
		VariationSeed:    "scorer-seed",
		Bars:             16,
	}
	melodySpec := offlineTemplate("melody", ctx)
	arpSpec := offlineTemplate("arpeggio", ctx)
	if melodySpec == nil || arpSpec == nil {
		t.Fatal("nil spec")
	}

	score := ScoreMelodyPhrase(melodySpec, arpSpec.Motif.Steps, prog, key)

	if score.PhraseScore < 0 || score.PhraseScore > 1 {
		t.Errorf("PhraseScore out of [0,1]: %.3f", score.PhraseScore)
	}
	if score.RestRatio < 0 || score.RestRatio > 1 {
		t.Errorf("RestRatio out of [0,1]: %.3f", score.RestRatio)
	}
	if score.PickupPresence < 0 || score.PickupPresence > 1 {
		t.Errorf("PickupPresence out of [0,1]: %.3f", score.PickupPresence)
	}
	if score.ContourScore < 0 || score.ContourScore > 1 {
		t.Errorf("ContourScore out of [0,1]: %.3f", score.ContourScore)
	}
	if score.RegisterSep < 0 || score.RegisterSep > 1 {
		t.Errorf("RegisterSep out of [0,1]: %.3f", score.RegisterSep)
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

func musicalFingerprint(steps []schema.StepSpec) string {
	var b strings.Builder
	for _, s := range steps {
		if !s.Active {
			b.WriteByte('-')
			continue
		}
		b.WriteString(theory.NoteNameOnly(s.Note))
		switch {
		case s.Accent:
			b.WriteByte('!')
		case s.Ghost:
			b.WriteByte('g')
		default:
			b.WriteByte('.')
		}
		if s.Legato {
			b.WriteByte('l')
		}
		if s.Staccato {
			b.WriteByte('s')
		}
		if s.Slide {
			b.WriteByte('>')
		}
		b.WriteByte('|')
	}
	return b.String()
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

func TestMelodyPhraseQuality(t *testing.T) {
	type keyCase struct{ root, scale, mode string }
	cases := []keyCase{
		{"A", "minor_natural", "minor"},
		{"D", "dorian", "dorian"},
		{"G", "mixolydian", "mixolydian"},
		{"E", "phrygian", "phrygian"},
	}
	for _, kc := range cases {
		t.Run(kc.root+"-"+kc.scale, func(t *testing.T) {
			key := theory.Key{Root: kc.root, Scale: kc.scale, Mode: kc.mode}
			prog := theory.SelectProgression(kc.root, kc.scale, "quality-seed")
			for seed := 1; seed <= 10; seed++ {
				ctx := MusicContext{
					BPM:              122,
					Key:              key,
					ChordProgression: prog,
					VariationSeed:    fmt.Sprintf("q-seed-%d", seed),
					Bars:             16,
				}
				melodySpec := offlineTemplate("melody", ctx)
				arpSpec := offlineTemplate("arpeggio", ctx)
				if melodySpec == nil || arpSpec == nil {
					t.Fatalf("seed %d: nil spec", seed)
				}
				score := ScoreMelodyPhrase(melodySpec, arpSpec.Motif.Steps, prog, key)
				if score.PhraseScore < 0.65 {
					t.Errorf("%s seed %d: PhraseScore=%.3f < 0.65 (rest=%.2f pickup=%.2f contour=%.2f chord=%.2f reg=%.2f)",
						kc.root, seed, score.PhraseScore,
						score.RestRatio, score.PickupPresence, score.ContourScore,
						score.ChordToneStrength, score.RegisterSep)
				}
				if score.RestRatio < 0.50 || score.RestRatio > 0.75 {
					t.Errorf("%s seed %d: RestRatio=%.3f outside [0.50, 0.75]", kc.root, seed, score.RestRatio)
				}
				if score.PickupPresence < 0.50 {
					t.Errorf("%s seed %d: PickupPresence=%.3f < 0.50", kc.root, seed, score.PickupPresence)
				}
			}
		})
	}
}

func TestMelodyNoUnresolvedLeaps(t *testing.T) {
	key := theory.Key{Root: "A", Scale: "minor_natural", Mode: "minor"}
	prog := theory.SelectProgression("A", "minor_natural", "leap-test")
	for seed := 1; seed <= 20; seed++ {
		ctx := MusicContext{
			BPM:              122,
			Key:              key,
			ChordProgression: prog,
			VariationSeed:    fmt.Sprintf("leap-seed-%d", seed),
			Bars:             16,
		}
		spec := offlineTemplate("melody", ctx)
		if spec == nil {
			t.Fatalf("seed %d: nil spec", seed)
		}
		numSections := len(spec.Motif.Steps) / stepsPerSection
		for sec := 0; sec < numSections; sec++ {
			base := sec * stepsPerSection
			if hasDoubleLeap(spec.Motif.Steps, base, base+stepsPerSection) {
				t.Errorf("seed %d section %d: double same-direction leap > 5 semitones", seed, sec)
			}
		}
	}
}

func TestListeningFixturesMelody(t *testing.T) {
	type Entry struct {
		Seed        string `json:"seed"`
		Key         string `json:"key"`
		Fingerprint string `json:"fingerprint"`
	}

	baselineRaw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "listening", "melody_before_fingerprints.json"))
	if err != nil {
		t.Skipf("baseline not found, skipping: %v", err)
	}
	var baseline []Entry
	if err := json.Unmarshal(baselineRaw, &baseline); err != nil {
		t.Fatalf("parse baseline: %v", err)
	}

	baselineMap := make(map[string]string, len(baseline))
	for _, e := range baseline {
		baselineMap[e.Key+"|"+e.Seed] = e.Fingerprint
	}

	seeds := []string{"fix-1", "fix-2", "fix-3"}
	keys := []struct{ root, scale, mode string }{
		{"A", "minor_natural", "minor"},
		{"D", "dorian", "dorian"},
	}

	changed := 0
	for _, kc := range keys {
		key := theory.Key{Root: kc.root, Scale: kc.scale, Mode: kc.mode}
		for _, seed := range seeds {
			prog := theory.SelectProgression(kc.root, kc.scale, seed)
			ctx := MusicContext{
				BPM: 122, Key: key, ChordProgression: prog,
				VariationSeed: seed, Bars: 16,
			}
			spec := offlineTemplate("melody", ctx)
			if spec == nil {
				t.Fatalf("%s %s: nil spec", kc.root, seed)
			}
			newFP := musicalFingerprint(spec.Motif.Steps)
			mapKey := kc.root + "-" + kc.scale + "|" + seed
			if oldFP, ok := baselineMap[mapKey]; ok && oldFP != newFP {
				changed++
			}
		}
	}
	if changed == 0 {
		t.Error("no melody fingerprints changed after refactor — phrase builder may not be wired correctly")
	}
	t.Logf("melody fingerprints changed: %d / %d", changed, len(seeds)*len(keys))
}
