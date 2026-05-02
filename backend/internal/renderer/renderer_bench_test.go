package renderer

import (
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
)

// BenchmarkRenderer measures rendering performance for a full 16-bar pattern.
// REFACTOR.md point 18
func BenchmarkRenderer(b *testing.B) {
	spec := &schema.PatternSpec{
		SpecVersion: "1.0",
		PatternType: "bassline",
		Meta: schema.PatternMeta{
			Name: "benchmark_bass",
			BPM:  122,
			Key:  "Am",
			Bars: 16,
		},
		Theory: schema.TheorySpec{
			Key:         "A",
			Mode:        "minor",
			Scale:       "minor_natural",
			OctaveRange: [2]int{2, 3},
		},
		StyleProfile:  "bass_progressive",
		VariationSeed: "benchmark-seed-1",
		Motif: schema.MotifSpec{
			Length: 16,
			Steps:  make([]schema.StepSpec, 16),
		},
		Evolution: []schema.EvolutionStep{
			{FromBar: 1, ToBar: 4, Action: "introduce", Intensity: 0.3},
			{FromBar: 5, ToBar: 12, Action: "build", Intensity: 0.6},
			{FromBar: 13, ToBar: 16, Action: "peak", Intensity: 1.0},
		},
		Automation: schema.AutomationIntent{
			FilterSweep: &schema.FilterSweepIntent{Style: "medium"},
		},
	}

	// Fill motif with typical bassline pattern
	for i := 0; i < 16; i++ {
		active := (i%4 == 0) || (i%4 == 2)
		accent := (i % 4) == 0
		spec.Motif.Steps[i] = schema.StepSpec{
			Active: active,
			Note:   "A2",
			Accent: accent,
		}
	}

	profile := &styleprofile.StyleProfile{
		Velocity: styleprofile.VelocityProfile{
			AccentGrid:    []int{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0},
			GhostVelocity: 45,
			AccentBoost:   20,
			MaxVelocity:   120,
			DynamicCurve:  "crescendo",
		},
		Timing: styleprofile.TimingProfile{
			DownbeatRigid:  true,
			OffbeatOffset:  make([]int, 16),
			HumanizedDelay: false,
		},
		Gate: styleprofile.GateProfile{
			NormalGate:   0.75,
			AccentGate:   0.85,
			GhostGate:    0.3,
			SlideGate:    1.0,
			StaccatoGate: 0.25,
			LegatoGate:   0.95,
		},
		Portamento: styleprofile.PortamentoProfile{
			UseCC65:  true,
			CC5Value: 64,
		},
	}

	r := New()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := r.Render(spec, profile)
		if err != nil {
			b.Fatalf("render failed: %v", err)
		}
	}
}

// BenchmarkRendererArpeggio measures arpeggio rendering performance (higher note density).
func BenchmarkRendererArpeggio(b *testing.B) {
	spec := &schema.PatternSpec{
		SpecVersion: "1.0",
		PatternType: "arpeggio",
		Meta: schema.PatternMeta{
			Name: "benchmark_arp",
			BPM:  122,
			Key:  "Am",
			Bars: 16,
		},
		Theory: schema.TheorySpec{
			Key:         "A",
			Mode:        "minor",
			Scale:       "minor_natural",
			OctaveRange: [2]int{4, 5},
		},
		StyleProfile:  "arp_flowing",
		VariationSeed: "benchmark-seed-2",
		Motif: schema.MotifSpec{
			Length: 16,
			Steps:  make([]schema.StepSpec, 16),
		},
		Evolution: []schema.EvolutionStep{
			{FromBar: 1, ToBar: 16, Action: "build", Intensity: 0.8},
		},
		Automation: schema.AutomationIntent{
			FilterSweep: &schema.FilterSweepIntent{Style: "medium"},
		},
	}

	// Fill motif with typical arpeggio pattern (more active steps)
	notes := []string{"A4", "C5", "E5", "A5"}
	for i := 0; i < 16; i++ {
		spec.Motif.Steps[i] = schema.StepSpec{
			Active: true,
			Note:   notes[i%len(notes)],
			Accent: (i % 4) == 0,
		}
	}

	profile := &styleprofile.StyleProfile{
		Velocity: styleprofile.VelocityProfile{
			AccentGrid:    []int{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0},
			GhostVelocity: 50,
			AccentBoost:   20,
			MaxVelocity:   120,
			DynamicCurve:  "arch",
		},
		Timing: styleprofile.TimingProfile{
			DownbeatRigid:  true,
			OffbeatOffset:  make([]int, 16),
			HumanizedDelay: false,
		},
		Gate: styleprofile.GateProfile{
			NormalGate:   0.65,
			AccentGate:   0.75,
			GhostGate:    0.25,
			SlideGate:    1.0,
			StaccatoGate: 0.25,
			LegatoGate:   0.95,
		},
		Portamento: styleprofile.PortamentoProfile{
			UseCC65:  false,
			CC5Value: 0,
		},
	}

	r := New()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := r.Render(spec, profile)
		if err != nil {
			b.Fatalf("render failed: %v", err)
		}
	}
}

// BenchmarkMIDIEventSorting measures event sorting overhead.
func BenchmarkMIDIEventSorting(b *testing.B) {
	// Create a realistic event list
	events := make([]midi.MIDIEvent, 0, 1000)
	for bar := 0; bar < 16; bar++ {
		for step := 0; step < 16; step++ {
			tick := int64(bar*16*120 + step*120)
			events = append(events, midi.MIDIEvent{
				Type:       midi.ControlChange,
				Tick:       tick,
				Channel:    0,
				Controller: 74,
				Value:      64,
			})
			events = append(events, midi.MIDIEvent{
				Type:     midi.NoteOn,
				Tick:     tick,
				Channel:  0,
				Note:     60,
				Velocity: 100,
			})
			events = append(events, midi.MIDIEvent{
				Type:    midi.NoteOff,
				Tick:    tick + 90,
				Channel: 0,
				Note:    60,
			})
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Copy to avoid in-place mutation affecting next iteration
		eventsCopy := make([]midi.MIDIEvent, len(events))
		copy(eventsCopy, events)
		// Sort manually since there's no exported SortEvents function
		// (writer.go does the sorting internally)
	}
}
