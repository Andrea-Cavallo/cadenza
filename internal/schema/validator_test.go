package schema

import "testing"

func validBasslineSpec() *PatternSpec {
	steps := make([]StepSpec, 16)
	activeNotes := []string{"A2", "A2", "E2", "E2", "G2", "G2", "D2", "D2", "A2", "A2"}
	for i := 0; i < 10; i++ {
		steps[i] = StepSpec{Active: true, Note: activeNotes[i]}
	}
	for i := 10; i < 16; i++ {
		steps[i] = StepSpec{Active: false}
	}

	return &PatternSpec{
		SpecVersion:  "1.0",
		PatternType:  "bassline",
		Meta:         PatternMeta{BPM: 122, Key: "Am", Bars: 16},
		Theory:       TheorySpec{Key: "A", Mode: "minor", Scale: "minor_natural", OctaveRange: [2]int{1, 3}},
		StyleProfile: "bass_progressive",
		Motif:        MotifSpec{Length: 16, Steps: steps},
		Evolution: []EvolutionStep{
			{FromBar: 1, ToBar: 4, Action: "introduce", Intensity: 0.3},
			{FromBar: 5, ToBar: 12, Action: "build", Intensity: 0.7},
			{FromBar: 13, ToBar: 16, Action: "peak", Intensity: 1.0},
		},
		Automation:    AutomationIntent{FilterSweep: &FilterSweepIntent{Style: "medium"}},
		VariationSeed: "test-seed",
	}
}

func TestValidator_ValidBassline(t *testing.T) {
	v := NewValidator()
	err := v.Validate(validBasslineSpec())
	if err != nil {
		t.Fatalf("valid bassline should pass: %v", err)
	}
}

func TestValidator_InvalidBPM(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	spec.Meta.BPM = 200
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("BPM 200 should fail validation")
	}
}

func TestValidator_NotesOutOfRange(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	spec.Motif.Steps[0] = StepSpec{Active: true, Note: "C7"}
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("C7 note in bassline should fail validation")
	}
}

func TestValidator_NotesOutOfScale(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	spec.Motif.Steps[0] = StepSpec{Active: true, Note: "C#2"}
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("C#2 in A minor should fail validation")
	}
}

func TestValidator_DensityTooLow(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	for i := range spec.Motif.Steps {
		spec.Motif.Steps[i] = StepSpec{Active: false}
	}
	spec.Motif.Steps[0] = StepSpec{Active: true, Note: "A2"}
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("only 1 active step should fail density check")
	}
}
