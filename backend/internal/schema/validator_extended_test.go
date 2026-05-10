package schema

import (
	"strings"
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

func TestValidator_UnknownPatternType(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	spec.PatternType = "nonexistent"
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("unknown pattern type should fail")
	}
	if !strings.Contains(err.Error(), "unknown pattern_type") {
		t.Fatalf("expected pattern_type error, got: %v", err)
	}
}

func TestValidator_UnknownStyleProfile(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	spec.StyleProfile = "bass_nonexistent"
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("unknown style profile should fail")
	}
	if !strings.Contains(err.Error(), "unknown style_profile") {
		t.Fatalf("expected style_profile error, got: %v", err)
	}
}

func TestValidator_InvalidEvolutionAction(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	spec.Evolution = []EvolutionStep{
		{FromBar: 1, ToBar: 16, Action: "nonexistent_action", Intensity: 0.5},
	}
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("unknown evolution action should fail")
	}
}

func TestValidator_InvalidEvolutionIntensity(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	spec.Evolution = []EvolutionStep{
		{FromBar: 1, ToBar: 16, Action: "build", Intensity: 1.5},
	}
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("intensity > 1 should fail")
	}
}

func TestValidator_DensityTooHigh(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	// Bassline max is 13 active steps — set all 16 to active
	for i := range spec.Motif.Steps {
		spec.Motif.Steps[i] = StepSpec{Active: true, Note: "A2"}
	}
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("16 active steps in bassline (max 13) should fail density check")
	}
}

func TestValidator_ActiveStepWithoutNote(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	spec.Motif.Steps[0] = StepSpec{Active: true, Note: ""}
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("active step without note should fail")
	}
}

func TestValidator_InvalidNoteName(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	spec.Motif.Steps[0] = StepSpec{Active: true, Note: "X9"}
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("invalid note name should fail")
	}
}

func TestValidator_InvalidBars(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	spec.Meta.Bars = 7
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("non-power-of-2 bars should fail")
	}
}

func TestValidator_ValidateWithBars(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	spec.Meta.Bars = 32
	err := v.ValidateWithBars(spec, 16)
	if err == nil {
		t.Fatal("bars mismatch (spec=32, expected=16) should fail")
	}
}

func TestValidator_ChordCoherenceBassline(t *testing.T) {
	v := NewValidator()
	spec := validBasslineSpec()
	// Set all notes to G2 which is not in Am chord (A, C, E)
	for i := range spec.Motif.Steps {
		if spec.Motif.Steps[i].Active {
			spec.Motif.Steps[i].Note = "G2"
		}
	}

	prog := theory.ChordProgression{
		Key:  "A",
		Mode: "minor",
		Chords: []theory.ProgressionChord{
			{Root: "A", Quality: "minor", Notes: []string{"A", "C", "E"}, Bars: [2]int{1, 4}},
			{Root: "F", Quality: "major", Notes: []string{"F", "A", "C"}, Bars: [2]int{5, 8}},
			{Root: "C", Quality: "major", Notes: []string{"C", "E", "G"}, Bars: [2]int{9, 12}},
			{Root: "G", Quality: "major", Notes: []string{"G", "B", "D"}, Bars: [2]int{13, 16}},
		},
	}

	err := v.ValidateWithChords(spec, prog)
	if err == nil {
		t.Fatal("G2 notes with Am chord should fail chord coherence for bassline")
	}
	if !strings.Contains(err.Error(), "chord coherence") {
		t.Fatalf("expected chord coherence error, got: %v", err)
	}
}

func TestValidator_ChordCoherencePasses(t *testing.T) {
	v := NewValidator()
	// Build a spec where each section's notes are predominantly chord tones
	steps := make([]StepSpec, 16)
	// Section 1 (steps 0-3): Am chord tones (A, C, E)
	steps[0] = StepSpec{Active: true, Note: "A2"}
	steps[1] = StepSpec{Active: true, Note: "C2"}
	steps[2] = StepSpec{Active: true, Note: "E2"}
	steps[3] = StepSpec{Active: false}
	// Section 2 (steps 4-7): Em chord tones (E, G, B)
	steps[4] = StepSpec{Active: true, Note: "E2"}
	steps[5] = StepSpec{Active: true, Note: "G2"}
	steps[6] = StepSpec{Active: true, Note: "B2"}
	steps[7] = StepSpec{Active: false}
	// Section 3 (steps 8-11): G chord tones (G, B, D)
	steps[8] = StepSpec{Active: true, Note: "G2"}
	steps[9] = StepSpec{Active: true, Note: "B2"}
	steps[10] = StepSpec{Active: true, Note: "D2"}
	steps[11] = StepSpec{Active: false}
	// Section 4 (steps 12-15): Dm chord tones (D, F, A)
	steps[12] = StepSpec{Active: true, Note: "D2"}
	steps[13] = StepSpec{Active: true, Note: "A2"}
	steps[14] = StepSpec{Active: false}
	steps[15] = StepSpec{Active: false}

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
			{Root: "A", Quality: "minor", Notes: []string{"A", "C", "E"}, Bars: [2]int{1, 4}},
			{Root: "E", Quality: "minor", Notes: []string{"E", "G", "B"}, Bars: [2]int{5, 8}},
			{Root: "G", Quality: "major", Notes: []string{"G", "B", "D"}, Bars: [2]int{9, 12}},
			{Root: "D", Quality: "minor", Notes: []string{"D", "F", "A"}, Bars: [2]int{13, 16}},
		},
	}

	err := v.ValidateWithChords(spec, prog)
	if err != nil {
		t.Fatalf("chord-coherent spec should pass: %v", err)
	}
}

func TestNewValidatorWithProfiles(t *testing.T) {
	v := NewValidatorWithProfiles([]string{"custom_profile"})
	spec := validBasslineSpec()
	spec.StyleProfile = "custom_profile"
	err := v.Validate(spec)
	if err != nil {
		t.Fatalf("custom profile should be accepted: %v", err)
	}
}

func TestNewValidatorWithProfiles_RejectsDefault(t *testing.T) {
	v := NewValidatorWithProfiles([]string{"only_this"})
	spec := validBasslineSpec()
	spec.StyleProfile = "bass_progressive"
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("bass_progressive should be rejected when custom profiles specified")
	}
}

func TestValidator_ValidMelody(t *testing.T) {
	v := NewValidator()
	steps := make([]StepSpec, 16)
	notes := []string{"A4", "B4", "C5", "D5", "E5"}
	for i := 0; i < 5; i++ {
		steps[i] = StepSpec{Active: true, Note: notes[i]}
	}
	for i := 5; i < 16; i++ {
		steps[i] = StepSpec{Active: false}
	}

	spec := &PatternSpec{
		PatternType:  "melody",
		Meta:         PatternMeta{BPM: 122, Key: "Am", Bars: 16},
		Theory:       TheorySpec{Key: "A", Mode: "minor", Scale: "minor_natural"},
		StyleProfile: "melody_expressive",
		Motif:        MotifSpec{Length: 16, Steps: steps},
		Evolution:    []EvolutionStep{{FromBar: 1, ToBar: 16, Action: "build", Intensity: 0.5}},
	}
	err := v.Validate(spec)
	if err != nil {
		t.Fatalf("valid melody should pass: %v", err)
	}
}

func TestValidator_ValidArpeggio(t *testing.T) {
	v := NewValidator()
	steps := make([]StepSpec, 16)
	notes := []string{"A3", "C4", "E4", "A4", "C4", "E4", "A3", "C4", "E4", "A4", "C4", "E4", "A3", "C4", "E4", "A4"}
	for i := 0; i < 16; i++ {
		steps[i] = StepSpec{Active: true, Note: notes[i]}
	}

	spec := &PatternSpec{
		PatternType:  "arpeggio",
		Meta:         PatternMeta{BPM: 122, Key: "Am", Bars: 16},
		Theory:       TheorySpec{Key: "A", Mode: "minor", Scale: "minor_natural"},
		StyleProfile: "arp_flowing",
		Motif:        MotifSpec{Length: 16, Steps: steps},
		Evolution:    []EvolutionStep{{FromBar: 1, ToBar: 16, Action: "build", Intensity: 0.5}},
	}
	err := v.Validate(spec)
	if err != nil {
		t.Fatalf("valid arpeggio should pass: %v", err)
	}
}

// validRollingBassSpec returns a PatternSpec for bass_rolling: 16 active steps, all in A minor scale.
func validRollingBassSpec() *PatternSpec {
	// 4 distinct 4-step phrases to satisfy anti-loop check
	notes := []string{
		"A2", "E2", "G2", "D2",
		"A2", "G2", "D2", "E2",
		"A2", "D2", "E2", "B1",
		"A2", "G2", "E2", "D2",
	}
	steps := make([]StepSpec, 16)
	for i, n := range notes {
		steps[i] = StepSpec{Active: true, Note: n}
	}
	return &PatternSpec{
		SpecVersion:  "1.0",
		PatternType:  "bassline",
		Meta:         PatternMeta{BPM: 122, Key: "Am", Bars: 4},
		Theory:       TheorySpec{Key: "A", Mode: "minor", Scale: "minor_natural"},
		StyleProfile: "bass_rolling",
		Motif:        MotifSpec{Length: 16, Steps: steps},
		Evolution:    []EvolutionStep{{FromBar: 1, ToBar: 4, Action: "build", Intensity: 0.5}},
	}
}

// TestValidator_RollingBass_DensityOverride verifies that 16 active steps pass for bass_rolling
// (regular bassline allows max 13; the style-profile override lifts this to 16).
func TestValidator_RollingBass_DensityOverride(t *testing.T) {
	v := NewValidator()
	spec := validRollingBassSpec()
	if err := v.Validate(spec); err != nil {
		t.Fatalf("rolling bass with 16 active steps should pass: %v", err)
	}
}

// TestValidator_RollingBass_ChromaticExceptionAllowed verifies that a single chromatic
// approach note (G#2, ±1 from G or A) not on the downbeat is accepted.
func TestValidator_RollingBass_ChromaticExceptionAllowed(t *testing.T) {
	v := NewValidator()
	spec := validRollingBassSpec()
	// G#2 = MIDI 44 — not in Am natural but ±1 from G2(43) and A2(45)
	spec.Motif.Steps[1] = StepSpec{Active: true, Note: "Ab2"} // Ab2 = G#2
	if err := v.Validate(spec); err != nil {
		t.Fatalf("single chromatic approach at non-downbeat should be allowed: %v", err)
	}
}

// TestValidator_RollingBass_ChromaticOnDownbeatFails verifies that a chromatic approach
// note placed on the section downbeat (step 0) is rejected.
func TestValidator_RollingBass_ChromaticOnDownbeatFails(t *testing.T) {
	v := NewValidator()
	spec := validRollingBassSpec()
	spec.Motif.Steps[0] = StepSpec{Active: true, Note: "Ab2"} // chromatic on downbeat
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("chromatic approach on downbeat should fail")
	}
	if !strings.Contains(err.Error(), "downbeat") {
		t.Fatalf("expected downbeat error, got: %v", err)
	}
}

// TestValidator_RollingBass_TooManyChromatics verifies that more than 2 chromatic
// approach notes in a single 16-step section are rejected.
func TestValidator_RollingBass_TooManyChromatics(t *testing.T) {
	v := NewValidator()
	spec := validRollingBassSpec()
	// Place 3 chromatic approach notes (all at non-downbeat positions)
	spec.Motif.Steps[1] = StepSpec{Active: true, Note: "Ab2"}
	spec.Motif.Steps[5] = StepSpec{Active: true, Note: "Ab2"}
	spec.Motif.Steps[9] = StepSpec{Active: true, Note: "Ab2"}
	err := v.Validate(spec)
	if err == nil {
		t.Fatal("3 chromatic notes in one section should fail")
	}
	if !strings.Contains(err.Error(), "exceed the 2-per-section limit") {
		t.Fatalf("expected per-section limit error, got: %v", err)
	}
}
