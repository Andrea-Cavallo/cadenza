package renderer

import (
	"github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// Renderer transforms a validated PatternSpec into MIDI events using a StyleProfile.
type Renderer struct{}

func New() *Renderer {
	return &Renderer{}
}

func (r *Renderer) Render(spec *schema.PatternSpec, profile *styleprofile.StyleProfile) ([]midi.MIDIEvent, error) {
	var events []midi.MIDIEvent

	// Portamento time CC5 at tick 0 for basslines with slide support.
	if profile.Portamento.UseCC65 {
		events = append(events, portamentoTimeEvent(profile))
	}

	// Filter sweep automation per bar.
	sweepStyle := ""
	if spec.Automation.FilterSweep != nil {
		sweepStyle = spec.Automation.FilterSweep.Style
	}

	for bar := 0; bar < spec.Meta.Bars; bar++ {
		// Apply evolution transforms for this bar.
		barSteps := applyEvolution(spec.Motif.Steps, bar, spec.Evolution)
		velScale := evolutionVelocityScale(bar, spec.Evolution)

		// Dynamic curve scaling based on bar position.
		velScale *= dynamicCurveScale(bar, spec.Meta.Bars, profile)

		// Filter sweep CC74 for this bar, synced to evolution.
		// REFACTOR.md point 5: Pass pattern type for phase offset calculation
		if sweepStyle != "" {
			events = append(events, filterSweepEvents(bar, spec.Meta.Bars, sweepStyle, spec.VariationSeed, spec.PatternType, profile, spec.Evolution)...)
		}

		for stepIdx, step := range barSteps {
			if !step.Active || step.Note == "" {
				continue
			}

			midiNote, err := theory.NoteToMIDI(step.Note)
			if err != nil {
				continue
			}

			tick := resolveTick(bar, stepIdx, profile)
			vel := resolveVelocity(stepIdx, step.Accent, step.Ghost, profile, velScale)

			// Slide gate: extend to next active note.
			var nextActiveTick int64
			if step.Slide {
				nextActiveTick = findNextActiveTick(barSteps, bar, stepIdx, profile)
			}
			gate := resolveGateTicksExt(stepIdx, step.Accent, step.Ghost, step.Slide, step.Legato, step.Staccato, profile, tick, nextActiveTick)

			// Portamento CC65 before each note.
			if profile.Portamento.UseCC65 {
				events = append(events, portamentoEvents(tick, step.Slide, profile)...)
			}

			events = append(events, midi.MIDIEvent{
				Type:     midi.NoteOn,
				Tick:     tick,
				Channel:  0,
				Note:     uint8(midiNote),
				Velocity: vel,
			})

			events = append(events, midi.MIDIEvent{
				Type:    midi.NoteOff,
				Tick:    tick + gate,
				Channel: 0,
				Note:    uint8(midiNote),
			})
		}
	}

	return events, nil
}

func dynamicCurveScale(bar, totalBars int, profile *styleprofile.StyleProfile) float64 {
	switch profile.Velocity.DynamicCurve {
	case "crescendo":
		return 0.7 + 0.3*float64(bar)/float64(totalBars-1)
	case "arch":
		mid := float64(totalBars) * 0.75
		if float64(bar) < mid {
			return 0.75 + 0.25*float64(bar)/mid
		}
		return 1.0 - 0.15*(float64(bar)-mid)/float64(float64(totalBars)-mid)
	case "plateau":
		// 0.7 → 1.0 (bars 1-6) → 1.0 (bars 7-12) → 0.85 (bars 13-16)
		riseEnd := float64(totalBars) * 0.375   // 6/16
		plateauEnd := float64(totalBars) * 0.75 // 12/16
		if float64(bar) < riseEnd {
			return 0.7 + 0.3*float64(bar)/riseEnd
		}
		if float64(bar) < plateauEnd {
			return 1.0
		}
		return 1.0 - 0.15*(float64(bar)-plateauEnd)/(float64(totalBars)-plateauEnd)
	case "tension":
		// 0.6 → 0.8 → 1.1 → 0.7 (tension with spike)
		quarter := float64(totalBars) / 4.0
		if float64(bar) < quarter {
			return 0.6 + 0.2*float64(bar)/quarter
		}
		if float64(bar) < 2*quarter {
			return 0.8 + 0.3*(float64(bar)-quarter)/quarter
		}
		if float64(bar) < 3*quarter {
			return 1.1 - 0.4*(float64(bar)-2*quarter)/quarter
		}
		return 0.7
	default:
		return 1.0
	}
}

func findNextActiveTick(steps []schema.StepSpec, bar, currentStep int, profile *styleprofile.StyleProfile) int64 {
	for i := currentStep + 1; i < len(steps); i++ {
		if steps[i].Active {
			return resolveTick(bar, i, profile)
		}
	}
	return resolveTick(bar+1, 0, profile)
}
