package renderer

import (
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
)

func testBasslineSpec() *schema.PatternSpec {
	steps := make([]schema.StepSpec, 16)
	notes := []string{"A2", "A2", "", "E2", "E2", "", "G2", "", "A2", "A2", "", "E2", "", "G2", "", ""}
	for i := 0; i < 16; i++ {
		if notes[i] != "" {
			steps[i] = schema.StepSpec{Active: true, Note: notes[i]}
		}
	}
	steps[0].Accent = true
	steps[8].Accent = true
	steps[3].Slide = true
	steps[13].Ghost = true

	return &schema.PatternSpec{
		SpecVersion:  "1.0",
		PatternType:  "bassline",
		Meta:         schema.PatternMeta{BPM: 122, Key: "Am", Bars: 16},
		Theory:       schema.TheorySpec{Key: "A", Mode: "minor", Scale: "minor_natural", OctaveRange: [2]int{1, 3}},
		StyleProfile: "bass_progressive",
		Motif:        schema.MotifSpec{Length: 16, Steps: steps},
		Evolution:    []schema.EvolutionStep{{FromBar: 1, ToBar: 16, Action: "build", Intensity: 0.5}},
		Automation:   schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: "medium"}},
	}
}

func TestRender_ProducesEvents(t *testing.T) {
	r := New()
	profile := &styleprofile.BassProgressive
	spec := testBasslineSpec()

	events, err := r.Render(spec, profile)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected MIDI events, got none")
	}

	var noteOns int
	for _, ev := range events {
		if ev.Type == midi.NoteOn {
			noteOns++
		}
	}
	if noteOns == 0 {
		t.Fatal("expected NoteOn events")
	}
}

func TestRender_VelocityClamped(t *testing.T) {
	r := New()
	profile := &styleprofile.BassProgressive
	spec := testBasslineSpec()

	events, err := r.Render(spec, profile)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	for _, ev := range events {
		if ev.Type == midi.NoteOn && ev.Velocity > 120 {
			t.Errorf("velocity %d exceeds max 120 at tick %d", ev.Velocity, ev.Tick)
		}
	}
}

func TestRender_NoteCount(t *testing.T) {
	r := New()
	profile := &styleprofile.BassProgressive
	spec := testBasslineSpec()

	events, err := r.Render(spec, profile)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	activeSteps := 0
	for _, s := range spec.Motif.Steps {
		if s.Active && s.Note != "" {
			activeSteps++
		}
	}

	var noteOns int
	for _, ev := range events {
		if ev.Type == midi.NoteOn {
			noteOns++
		}
	}

	// 16 bars × active steps per bar
	expected := spec.Meta.Bars * activeSteps
	if noteOns != expected {
		t.Errorf("expected %d NoteOn events (bars=%d × steps=%d), got %d",
			expected, spec.Meta.Bars, activeSteps, noteOns)
	}
}

func TestRender_EvolutionIntroduce_ReducesDensity(t *testing.T) {
	r := New()
	profile := &styleprofile.BassProgressive
	spec := testBasslineSpec()
	spec.Evolution = []schema.EvolutionStep{
		{FromBar: 1, ToBar: 16, Action: "introduce", Intensity: 0.4},
	}

	events, err := r.Render(spec, profile)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var noteOns int
	for _, ev := range events {
		if ev.Type == midi.NoteOn {
			noteOns++
		}
	}

	activeSteps := 0
	for _, s := range spec.Motif.Steps {
		if s.Active && s.Note != "" {
			activeSteps++
		}
	}
	fullCount := spec.Meta.Bars * activeSteps

	if noteOns >= fullCount {
		t.Errorf("introduce should reduce density: got %d NoteOns, full would be %d", noteOns, fullCount)
	}
	minExpected := spec.Meta.Bars * minActiveFloor
	if noteOns < minExpected {
		t.Errorf("introduce should not go below floor: got %d NoteOns, min expected %d", noteOns, minExpected)
	}
}

func testSpecWithInactiveNotes() *schema.PatternSpec {
	steps := make([]schema.StepSpec, 16)
	notes := []string{"A2", "A2", "G2", "E2", "E2", "D2", "G2", "A2", "A2", "A2", "G2", "E2", "D2", "G2", "E2", "A2"}
	for i := 0; i < 16; i++ {
		steps[i] = schema.StepSpec{Active: i%2 == 0, Note: notes[i]}
	}
	steps[0].Accent = true
	steps[8].Accent = true

	return &schema.PatternSpec{
		SpecVersion:  "1.0",
		PatternType:  "bassline",
		Meta:         schema.PatternMeta{BPM: 122, Key: "Am", Bars: 16},
		Theory:       schema.TheorySpec{Key: "A", Mode: "minor", Scale: "minor_natural", OctaveRange: [2]int{1, 3}},
		StyleProfile: "bass_progressive",
		Motif:        schema.MotifSpec{Length: 16, Steps: steps},
		Evolution:    []schema.EvolutionStep{{FromBar: 1, ToBar: 16, Action: "build", Intensity: 0.5}},
		Automation:   schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: "medium"}},
	}
}

func TestRender_EvolutionPeak_IncreasesDensity(t *testing.T) {
	r := New()
	profile := &styleprofile.BassProgressive
	spec := testSpecWithInactiveNotes()

	buildEvents, err := r.Render(spec, profile)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	var buildNoteOns int
	for _, ev := range buildEvents {
		if ev.Type == midi.NoteOn {
			buildNoteOns++
		}
	}

	spec.Evolution = []schema.EvolutionStep{
		{FromBar: 1, ToBar: 16, Action: "peak", Intensity: 0.9},
	}
	peakEvents, err := r.Render(spec, profile)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	var peakNoteOns int
	for _, ev := range peakEvents {
		if ev.Type == midi.NoteOn {
			peakNoteOns++
		}
	}

	if peakNoteOns <= buildNoteOns {
		t.Errorf("peak should increase density: peak=%d, build=%d", peakNoteOns, buildNoteOns)
	}
}

func TestRender_EvolutionRelease_ReducesDensity(t *testing.T) {
	r := New()
	profile := &styleprofile.BassProgressive
	spec := testBasslineSpec()
	spec.Evolution = []schema.EvolutionStep{
		{FromBar: 1, ToBar: 16, Action: "release", Intensity: 0.6},
	}

	events, err := r.Render(spec, profile)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var noteOns int
	for _, ev := range events {
		if ev.Type == midi.NoteOn {
			noteOns++
		}
	}

	activeSteps := 0
	for _, s := range spec.Motif.Steps {
		if s.Active && s.Note != "" {
			activeSteps++
		}
	}
	fullCount := spec.Meta.Bars * activeSteps

	if noteOns >= fullCount {
		t.Errorf("release should reduce density: got %d NoteOns, full would be %d", noteOns, fullCount)
	}
}
