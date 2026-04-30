package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

func TestSpecIO_RoundTripYAMLAndJSON(t *testing.T) {
	spec := validBasslineSpec()
	yamlPath := filepath.Join(t.TempDir(), "specs", "bass.yaml")
	jsonPath := filepath.Join(t.TempDir(), "specs", "bass.json")

	if err := DumpToYAML(spec, yamlPath); err != nil {
		t.Fatalf("DumpToYAML error: %v", err)
	}
	loadedYAML, err := LoadFromYAML(yamlPath)
	if err != nil {
		t.Fatalf("LoadFromYAML error: %v", err)
	}
	if loadedYAML.PatternType != spec.PatternType || loadedYAML.Meta.Key != spec.Meta.Key {
		t.Fatalf("yaml round trip mismatch: got %+v", loadedYAML.Meta)
	}

	if err := DumpToJSON(spec, jsonPath); err != nil {
		t.Fatalf("DumpToJSON error: %v", err)
	}
	loadedJSON, err := LoadFromJSON(jsonPath)
	if err != nil {
		t.Fatalf("LoadFromJSON error: %v", err)
	}
	if loadedJSON.StyleProfile != spec.StyleProfile || loadedJSON.VariationSeed != spec.VariationSeed {
		t.Fatalf("json round trip mismatch: got %+v", loadedJSON)
	}
}

func TestSpecIO_InvalidInputsFail(t *testing.T) {
	tmp := t.TempDir()
	badYAML := filepath.Join(tmp, "bad.yaml")
	badJSON := filepath.Join(tmp, "bad.json")

	if err := DumpToYAML(validBasslineSpec(), filepath.Join(tmp, "out", "ok.yaml")); err != nil {
		t.Fatalf("unexpected yaml dump error: %v", err)
	}

	if err := DumpToJSON(validBasslineSpec(), filepath.Join(tmp, "out", "ok.json")); err != nil {
		t.Fatalf("unexpected json dump error: %v", err)
	}

	if err := os.WriteFile(badYAML, []byte("not: [valid"), 0o644); err != nil {
		t.Fatalf("write bad yaml: %v", err)
	}
	if err := os.WriteFile(badJSON, []byte("{bad json"), 0o644); err != nil {
		t.Fatalf("write bad json: %v", err)
	}

	if _, err := LoadFromYAML(filepath.Join(tmp, "missing.yaml")); err == nil {
		t.Fatal("expected missing YAML file to fail")
	}
	if _, err := LoadFromJSON(filepath.Join(tmp, "missing.json")); err == nil {
		t.Fatal("expected missing JSON file to fail")
	}
	if _, err := LoadFromYAML(badYAML); err == nil {
		t.Fatal("expected invalid YAML to fail")
	}
	if _, err := LoadFromJSON(badJSON); err == nil {
		t.Fatal("expected invalid JSON to fail")
	}
}

func TestGenerateJSONSchema_HasCoreFields(t *testing.T) {
	properties, required, err := GenerateJSONSchema()
	if err != nil {
		t.Fatalf("GenerateJSONSchema error: %v", err)
	}

	for _, key := range []string{"pattern_type", "meta", "theory", "motif", "variation_seed"} {
		if _, ok := properties[key]; !ok {
			t.Fatalf("missing property %q", key)
		}
	}

	if len(required) == 0 {
		t.Fatal("expected required fields")
	}
}

func TestValidator_ValidateWithChordsAndBars(t *testing.T) {
	v := NewValidator()
	steps := make([]StepSpec, 16)
	steps[0] = StepSpec{Active: true, Note: "A2"}
	steps[1] = StepSpec{Active: true, Note: "C2"}
	steps[2] = StepSpec{Active: true, Note: "E2"}
	steps[4] = StepSpec{Active: true, Note: "E2"}
	steps[5] = StepSpec{Active: true, Note: "G2"}
	steps[6] = StepSpec{Active: true, Note: "B2"}
	steps[8] = StepSpec{Active: true, Note: "G2"}
	steps[9] = StepSpec{Active: true, Note: "B2"}
	steps[10] = StepSpec{Active: true, Note: "D2"}
	steps[12] = StepSpec{Active: true, Note: "D2"}
	steps[13] = StepSpec{Active: true, Note: "F2"}
	steps[14] = StepSpec{Active: true, Note: "A2"}

	spec := &PatternSpec{
		PatternType:  "bassline",
		Meta:         PatternMeta{BPM: 122, Key: "Am", Bars: 16},
		Theory:       TheorySpec{Key: "A", Mode: "minor", Scale: "minor_natural"},
		StyleProfile: "bass_progressive",
		Motif:        MotifSpec{Length: 16, Steps: steps},
		Evolution:    []EvolutionStep{{FromBar: 1, ToBar: 16, Action: "build", Intensity: 0.5}},
	}
	prog := theory.ChordProgression{
		Key:  "A",
		Mode: "minor",
		Chords: []theory.ProgressionChord{
			{Root: "A", Quality: "minor", Bars: [2]int{1, 4}},
			{Root: "E", Quality: "minor", Bars: [2]int{5, 8}},
			{Root: "G", Quality: "major", Bars: [2]int{9, 12}},
			{Root: "D", Quality: "minor", Bars: [2]int{13, 16}},
		},
	}

	if err := v.ValidateWithChordsAndBars(spec, prog, 16); err != nil {
		t.Fatalf("ValidateWithChordsAndBars should pass: %v", err)
	}
}

func TestValidator_HelperBranches(t *testing.T) {
	if qualitySuffixValidator("minor") != "m" {
		t.Fatal("minor suffix mismatch")
	}
	if qualitySuffixValidator("dim") != "dim" {
		t.Fatal("dim suffix mismatch")
	}
	if qualitySuffixValidator("aug") != "aug" {
		t.Fatal("aug suffix mismatch")
	}
	if qualitySuffixValidator("major") != "" {
		t.Fatal("major suffix should be empty")
	}

	if !isChordTone("A", []string{"A", "C", "E"}) {
		t.Fatal("expected A to be chord tone")
	}
	if isChordTone("B", []string{"A", "C", "E"}) {
		t.Fatal("expected B not to be chord tone")
	}

	active, hits := chordToneStats([]StepSpec{
		{Active: true, Note: "A2"},
		{Active: true, Note: "B2"},
		{Active: false, Note: "C2"},
	}, []string{"A", "C", "E"})
	if active != 2 || hits != 1 {
		t.Fatalf("unexpected chordToneStats: active=%d hits=%d", active, hits)
	}

	if !meetsChordThreshold(3, 4, 0.75) {
		t.Fatal("expected threshold to pass")
	}
	if meetsChordThreshold(2, 4, 0.75) {
		t.Fatal("expected threshold to fail")
	}
}
