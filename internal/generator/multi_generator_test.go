package generator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/llm"
	midipkg "github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
)

func mockBasslineJSON() []byte {
	steps := make([]schema.StepSpec, 16)
	notes := []string{"A2", "A2", "E2", "E2", "G2", "G2", "D2", "D2", "A2", "E2", "", "", "", "", "", ""}
	for i, n := range notes {
		if n != "" {
			steps[i] = schema.StepSpec{Active: true, Note: n}
		}
	}
	steps[0].Accent = true
	steps[8].Accent = true

	spec := schema.PatternSpec{
		SpecVersion:  "1.0",
		PatternType:  "bassline",
		Meta:         schema.PatternMeta{Name: "test", BPM: 122, Key: "Am", Bars: 16},
		Theory:       schema.TheorySpec{Key: "A", Mode: "minor", Scale: "minor_natural", OctaveRange: [2]int{1, 3}},
		StyleProfile: "bass_progressive",
		Motif:        schema.MotifSpec{Length: 16, Steps: steps},
		Evolution:    []schema.EvolutionStep{{FromBar: 1, ToBar: 16, Action: "sustain", Intensity: 0.5}},
		Automation:   schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: "medium"}},
	}
	b, _ := json.Marshal(spec)
	return b
}

func newTestMultiGenerator(t *testing.T, outDir string) *MultiGenerator {
	t.Helper()
	mock := &llm.MockProvider{Response: mockBasslineJSON()}
	v := schema.NewValidator()
	reg := styleprofile.NewRegistry()
	rend := renderer.New()
	w := midipkg.NewWriter(122.0)
	mg := NewMultiGenerator(mock, v, reg, rend, w, outDir)
	mg.NoLLM = true
	return mg
}

func TestMultiGenerator_ProducesThreeFiles(t *testing.T) {
	outDir := t.TempDir()
	mg := newTestMultiGenerator(t, outDir)

	result, err := mg.Generate(context.Background(), 122.0, "Am")
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if len(result.Files) == 0 {
		t.Fatal("expected output files, got none")
	}

	for _, f := range result.Files {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("output file %q not found: %v", f, err)
		}
	}
}

func TestMultiGenerator_FilesHaveMIDIExtension(t *testing.T) {
	outDir := t.TempDir()
	mg := newTestMultiGenerator(t, outDir)

	result, err := mg.Generate(context.Background(), 122.0, "Am")
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	for _, f := range result.Files {
		if filepath.Ext(f) != ".mid" {
			t.Errorf("expected .mid extension, got %q", filepath.Ext(f))
		}
	}
}

func TestMultiGenerator_FilesAreNonEmpty(t *testing.T) {
	outDir := t.TempDir()
	mg := newTestMultiGenerator(t, outDir)

	result, err := mg.Generate(context.Background(), 122.0, "Am")
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	for _, f := range result.Files {
		info, err := os.Stat(f)
		if err != nil {
			t.Errorf("stat %q: %v", f, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("file %q is empty", f)
		}
	}
}

func TestMultiGenerator_VariationSeedSet(t *testing.T) {
	outDir := t.TempDir()
	mg := newTestMultiGenerator(t, outDir)

	result, err := mg.Generate(context.Background(), 122.0, "Am")
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if result.VariationSeed == "" {
		t.Error("variation seed must not be empty")
	}
}

func TestMultiGenerator_InvalidBPMFails(t *testing.T) {
	outDir := t.TempDir()
	mg := newTestMultiGenerator(t, outDir)

	_, err := mg.Generate(context.Background(), 200.0, "Am")
	if err == nil {
		t.Error("expected error for BPM=200, got nil")
	}
}

func TestMultiGenerator_InvalidKeyFails(t *testing.T) {
	outDir := t.TempDir()
	mg := newTestMultiGenerator(t, outDir)

	_, err := mg.Generate(context.Background(), 122.0, "ZZZ")
	if err == nil {
		t.Error("expected error for invalid key, got nil")
	}
}
