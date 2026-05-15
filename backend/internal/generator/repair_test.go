package generator

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// TestRepairChordCoherenceEnharmonic verifies that repairChordCoherence writes chord tones
// using their flat/sharp spelling from the chord, not via MIDIToNote which always uses sharps.
// Regression for: Fm chord tones = ["F","Ab","C"] but MIDIToNote(44) = "G#2" ≠ "Ab".
func TestRepairChordCoherenceEnharmonic(t *testing.T) {
	// Key: Cm, scale: minor_natural.
	// Progression: Cm → Fm → Bb → Eb (4 chords, 16 bars).
	// Section 3 = Fm. Fm chord tones: F, Ab, C.
	// Bassline has 3 active steps in section 3, none are chord tones (uses G2, D2, Eb2).
	// Diatonic Cm progression: i–iv–VII–III (all notes in Cm natural minor)
	prog := theory.ChordProgression{
		Chords: []theory.ProgressionChord{
			{Root: "C", Quality: "minor", Bars: [2]int{1, 4}},
			{Root: "F", Quality: "minor", Bars: [2]int{5, 8}},
			{Root: "Bb", Quality: "major", Bars: [2]int{9, 12}}, // VII = Bb major
			{Root: "Eb", Quality: "major", Bars: [2]int{13, 16}}, // III = Eb major
		},
	}

	spec := &schema.PatternSpec{
		SpecVersion:  "1.0",
		PatternType:  "bassline",
		StyleProfile: "bass_progressive",
		Meta:         schema.PatternMeta{BPM: 122, Bars: 16},
		Theory:       schema.TheorySpec{Key: "C", Scale: "minor_natural"},
		Evolution: []schema.EvolutionStep{
			{FromBar: 1, ToBar: 4, Action: "introduce", Intensity: 0.3},
			{FromBar: 5, ToBar: 8, Action: "build", Intensity: 0.6},
			{FromBar: 9, ToBar: 12, Action: "peak", Intensity: 0.9},
			{FromBar: 13, ToBar: 16, Action: "release", Intensity: 0.5},
		},
		Motif: schema.MotifSpec{
			Length: 16,
			Steps: func() []schema.StepSpec {
				steps := make([]schema.StepSpec, 16)
				// Sections 1,2,4: valid chord tones so coherence passes.
				// Cm tones: C, Eb, G
				steps[0] = schema.StepSpec{Active: true, Note: "C2", Accent: true}
				steps[1] = schema.StepSpec{Active: true, Note: "G2"}
				steps[2] = schema.StepSpec{Active: true, Note: "Eb2"}
				steps[3] = schema.StepSpec{Active: false}
				// Fm tones: F, Ab, C — use wrong notes (G, D, Eb = NOT Fm chord tones)
				steps[4] = schema.StepSpec{Active: true, Note: "G2"} // not Fm
				steps[5] = schema.StepSpec{Active: true, Note: "D2"} // not Fm
				steps[6] = schema.StepSpec{Active: true, Note: "Eb2"} // not Fm
				steps[7] = schema.StepSpec{Active: false}
				// Bb tones: Bb, D, F
				steps[8] = schema.StepSpec{Active: true, Note: "Bb2", Accent: true}
				steps[9] = schema.StepSpec{Active: true, Note: "D2"}
				steps[10] = schema.StepSpec{Active: true, Note: "F2"}
				steps[11] = schema.StepSpec{Active: false}
				// Eb tones: Eb, G, Bb
				steps[12] = schema.StepSpec{Active: true, Note: "Eb2", Accent: true}
				steps[13] = schema.StepSpec{Active: true, Note: "G2"}
				steps[14] = schema.StepSpec{Active: true, Note: "Bb2"}
				steps[15] = schema.StepSpec{Active: false}
				return steps
			}(),
		},
	}

	c := repairConstraints["bassline"]
	repairChordCoherence(spec, prog, c)

	// After repair, steps 4-6 should be Fm chord tones: F, Ab, or C.
	fmTones := map[string]bool{"F": true, "Ab": true, "C": true}
	for i := 4; i <= 6; i++ {
		step := spec.Motif.Steps[i]
		if !step.Active || step.Note == "" {
			continue
		}
		name := theory.NoteNameOnly(step.Note)
		if !fmTones[name] {
			t.Errorf("step[%d] = %q: expected Fm chord tone (F, Ab, C), got %q", i, step.Note, name)
		}
		// Critical: must NOT be spelled as sharp enharmonic equivalent.
		if strings.Contains(step.Note, "G#") {
			t.Errorf("step[%d] = %q: got G# instead of Ab — enharmonic spelling bug", i, step.Note)
		}
	}
}

// TestNearestChordToneNoteSpelling verifies the note name spelling is preserved.
func TestNearestChordToneNoteSpelling(t *testing.T) {
	c := repairConstraints["bassline"] // [33, 55, 8, 13]

	tests := []struct {
		midi       int
		chordTones []string
		wantName   string // expected note name component (no octave)
	}{
		{midi: 43, chordTones: []string{"F", "Ab", "C"}, wantName: "Ab"}, // G2=43, nearest is Ab2=44
		{midi: 41, chordTones: []string{"F", "Ab", "C"}, wantName: "F"},  // F2=41, exact match
		{midi: 48, chordTones: []string{"F", "Ab", "C"}, wantName: "C"},  // C3=48, exact match
	}

	for _, tt := range tests {
		got := nearestChordToneNote(tt.midi, tt.chordTones, c)
		if got == "" {
			t.Errorf("midi=%d chords=%v: got empty string", tt.midi, tt.chordTones)
			continue
		}
		gotName := theory.NoteNameOnly(got)
		if gotName != tt.wantName {
			t.Errorf("midi=%d chords=%v: got name %q, want %q (full note: %q)", tt.midi, tt.chordTones, gotName, tt.wantName, got)
		}
	}
}

// TestRepairSpecFmChordCoherence is an integration test for the full repairSpec path
// simulating what happens when DeepSeek returns a bassline with 0% chord coherence on Fm section.
func TestRepairSpecFmChordCoherence(t *testing.T) {
	musicCtx := MusicContext{
		BPM:  122,
		Key:  theory.Key{Root: "C", Mode: "minor", Scale: "minor_natural"},
		Bars: 16,
		// Diatonic Cm progression: i–iv–VII–III
		ChordProgression: theory.ChordProgression{
			Chords: []theory.ProgressionChord{
				{Root: "C", Quality: "minor", Bars: [2]int{1, 4}},
				{Root: "F", Quality: "minor", Bars: [2]int{5, 8}},
				{Root: "Bb", Quality: "major", Bars: [2]int{9, 12}}, // VII = Bb major
				{Root: "Eb", Quality: "major", Bars: [2]int{13, 16}}, // III = Eb major
			},
		},
	}

	// Build a spec that only fails chord coherence on Fm (section 2, index 1).
	spec := schema.PatternSpec{
		SpecVersion:  "1.0",
		PatternType:  "bassline",
		StyleProfile: "bass_progressive",
		Meta:         schema.PatternMeta{BPM: 122, Bars: 16},
		Theory:       schema.TheorySpec{Key: "C", Scale: "minor_natural"},
		Evolution: []schema.EvolutionStep{
			{FromBar: 1, ToBar: 4, Action: "introduce", Intensity: 0.3},
			{FromBar: 5, ToBar: 8, Action: "build", Intensity: 0.6},
			{FromBar: 9, ToBar: 12, Action: "peak", Intensity: 0.9},
			{FromBar: 13, ToBar: 16, Action: "release", Intensity: 0.5},
		},
		Motif: schema.MotifSpec{
			Length: 16,
			Steps: []schema.StepSpec{
				// Cm section: C, Eb, G (chord tones of Cm)
				{Active: true, Note: "C2", Accent: true},
				{Active: true, Note: "Eb2"},
				{Active: true, Note: "G2"},
				{Active: false},
				// Fm section: use D, G, Eb (NOT Fm chord tones)
				{Active: true, Note: "D2", Accent: true},
				{Active: true, Note: "G2"},
				{Active: true, Note: "Eb2"},
				{Active: false},
				// Bb section: Bb, D, F (chord tones of Bbm)
				{Active: true, Note: "Bb2", Accent: true},
				{Active: true, Note: "D2"},
				{Active: true, Note: "F2"},
				{Active: false},
				// Eb section: Eb, G, Bb (chord tones of Eb major)
				{Active: true, Note: "Eb2", Accent: true},
				{Active: true, Note: "G2"},
				{Active: true, Note: "Bb2"},
				{Active: false},
			},
		},
	}

	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}

	v := schema.NewValidator()
	validate := func(b []byte) error {
		var s schema.PatternSpec
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		s.Theory.Key = musicCtx.Key.Root
		s.Theory.Scale = musicCtx.Key.Scale
		return v.ValidateWithChords(&s, musicCtx.ChordProgression)
	}

	repaired, err := repairSpec(raw, musicCtx, validate)
	if err != nil {
		t.Fatalf("repairSpec failed: %v", err)
	}

	var repairedSpec schema.PatternSpec
	if err := json.Unmarshal(repaired, &repairedSpec); err != nil {
		t.Fatalf("unmarshal repaired: %v", err)
	}

	// Fm section is steps 4-7. After repair, steps 4-6 must be Fm chord tones.
	fmTones := map[string]bool{"F": true, "Ab": true, "C": true}
	for i := 4; i <= 6; i++ {
		step := repairedSpec.Motif.Steps[i]
		if !step.Active {
			continue
		}
		name := theory.NoteNameOnly(step.Note)
		if !fmTones[name] {
			t.Errorf("after repair, step[%d] = %q is not an Fm chord tone", i, step.Note)
		}
	}
}
