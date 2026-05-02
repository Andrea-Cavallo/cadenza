package schema

import (
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// BenchmarkValidator measures validation performance for a full PatternSpec.
// REFACTOR.md point 18
func BenchmarkValidator(b *testing.B) {
	spec := &PatternSpec{
		SpecVersion: "1.0",
		PatternType: "bassline",
		Meta: PatternMeta{
			Name: "benchmark_bass",
			BPM:  122,
			Key:  "Am",
			Bars: 16,
		},
		Theory: TheorySpec{
			Key:         "A",
			Mode:        "minor",
			Scale:       "minor_natural",
			OctaveRange: [2]int{2, 3},
		},
		StyleProfile:  "bass_progressive",
		VariationSeed: "benchmark-seed-1",
		Motif: MotifSpec{
			Length: 16,
			Steps:  make([]StepSpec, 16),
		},
		Evolution: []EvolutionStep{
			{FromBar: 1, ToBar: 4, Action: "introduce", Intensity: 0.3},
			{FromBar: 5, ToBar: 12, Action: "build", Intensity: 0.6},
			{FromBar: 13, ToBar: 16, Action: "peak", Intensity: 1.0},
		},
		Automation: AutomationIntent{
			FilterSweep: &FilterSweepIntent{Style: "medium"},
		},
	}

	// Fill motif with typical bassline pattern
	notes := []string{"A2", "A2", "E2", "A2", "C3", "A2", "E2", "A2",
		"A2", "G2", "E2", "A2", "A2", "A2", "C3", "A2"}
	for i := 0; i < 16; i++ {
		active := (i%4 == 0) || (i%4 == 2)
		accent := (i % 4) == 0
		spec.Motif.Steps[i] = StepSpec{
			Active: active,
			Note:   notes[i],
			Accent: accent,
		}
	}

	// Create chord progression
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

	v := NewValidator()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := v.ValidateWithChords(spec, prog)
		if err != nil {
			b.Fatalf("validation failed: %v", err)
		}
	}
}

// BenchmarkValidatorArpeggio measures validation performance for high-density arpeggio.
func BenchmarkValidatorArpeggio(b *testing.B) {
	spec := &PatternSpec{
		SpecVersion: "1.0",
		PatternType: "arpeggio",
		Meta: PatternMeta{
			Name: "benchmark_arp",
			BPM:  122,
			Key:  "Am",
			Bars: 16,
		},
		Theory: TheorySpec{
			Key:         "A",
			Mode:        "minor",
			Scale:       "minor_natural",
			OctaveRange: [2]int{4, 5},
		},
		StyleProfile:  "arp_flowing",
		VariationSeed: "benchmark-seed-2",
		Motif: MotifSpec{
			Length: 16,
			Steps:  make([]StepSpec, 16),
		},
		Evolution: []EvolutionStep{
			{FromBar: 1, ToBar: 16, Action: "build", Intensity: 0.8},
		},
		Automation: AutomationIntent{
			FilterSweep: &FilterSweepIntent{Style: "medium"},
		},
	}

	// Fill motif with typical arpeggio pattern (all steps active, all in scale)
	notes := []string{"A4", "C5", "E5", "A5", "C5", "E5", "A4", "C5",
		"A4", "C5", "E5", "C5", "A4", "C5", "E5", "A5"}
	for i := 0; i < 16; i++ {
		spec.Motif.Steps[i] = StepSpec{
			Active: true,
			Note:   notes[i],
			Accent: (i % 4) == 0,
		}
	}

	v := NewValidator()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use Validate() instead of ValidateWithChords() to skip chord coherence check
		err := v.Validate(spec)
		if err != nil {
			b.Fatalf("validation failed: %v", err)
		}
	}
}

// BenchmarkChordCoherenceCheck isolates chord coherence validation overhead.
func BenchmarkChordCoherenceCheck(b *testing.B) {
	spec := &PatternSpec{
		SpecVersion:  "1.0",
		PatternType:  "bassline",
		Meta:         PatternMeta{BPM: 122, Key: "Am", Bars: 16},
		Theory:       TheorySpec{Key: "A", Mode: "minor", Scale: "minor_natural", OctaveRange: [2]int{2, 3}},
		StyleProfile: "bass_progressive",
		Motif: MotifSpec{
			Length: 16,
			Steps: []StepSpec{
				{Active: true, Note: "A2"},
				{Active: true, Note: "A2"},
				{Active: true, Note: "E2"},
				{Active: true, Note: "A2"},
				{Active: true, Note: "C3"},
				{Active: true, Note: "A2"},
				{Active: true, Note: "E2"},
				{Active: true, Note: "A2"},
				{Active: true, Note: "A2"},
				{Active: true, Note: "G2"},
				{Active: true, Note: "E2"},
				{Active: true, Note: "A2"},
				{Active: true, Note: "A2"},
				{Active: true, Note: "A2"},
				{Active: true, Note: "C3"},
				{Active: true, Note: "A2"},
			},
		},
		Evolution: []EvolutionStep{
			{FromBar: 1, ToBar: 16, Action: "build", Intensity: 0.8},
		},
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

	v := NewValidator()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = v.ValidateWithChords(spec, prog)
	}
}
