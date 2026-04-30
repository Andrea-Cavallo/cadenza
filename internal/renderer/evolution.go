package renderer

import (
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
)

const minActiveFloor = 4

func applyEvolution(steps []schema.StepSpec, bar int, evolutions []schema.EvolutionStep) []schema.StepSpec {
	return buildBarSteps(steps, bar, evolutions)
}

func evolutionVelocityScale(bar int, evolutions []schema.EvolutionStep) float64 {
	barNum := bar + 1
	for _, ev := range evolutions {
		if barNum < ev.FromBar || barNum > ev.ToBar {
			continue
		}
		switch ev.Action {
		case "introduce":
			return 0.8 * ev.Intensity / 0.3
		case "build":
			return 1.0
		case "peak":
			return 1.0 + 0.1*ev.Intensity/0.8
		case "release":
			return 0.9
		}
	}
	return 1.0
}

func buildBarSteps(steps []schema.StepSpec, bar int, evolutions []schema.EvolutionStep) []schema.StepSpec {
	result := make([]schema.StepSpec, len(steps))
	copy(result, steps)

	barNum := bar + 1

	for _, ev := range evolutions {
		if barNum < ev.FromBar || barNum > ev.ToBar {
			continue
		}

		switch ev.Action {
		case "octave_up":
			applyOctaveShift(result, 1, ev.Intensity)
		case "octave_down":
			applyOctaveShift(result, -1, ev.Intensity)
		case "introduce":
			current := countActive(result)
			targetActive := int(float64(current) * 0.6)
			if targetActive < minActiveFloor {
				targetActive = minActiveFloor
			}
			if targetActive > current {
				targetActive = current
			}
			deactivateSteps(result, current-targetActive)
		case "build":
			// REFACTOR.md point 5: build adds fill notes (ghost notes on off-beat)
			addFillNotes(result, int(ev.Intensity*3))
		case "peak":
			// REFACTOR.md point 5: peak adds octave-up on selected notes + velocity push (handled by velocity scale)
			activateSteps(result, 2)
			applyOctaveShift(result, 1, 0.3) // 30% of notes go up an octave
		case "release":
			// REFACTOR.md point 5: release removes fills, returns to base motif with fewer accents
			removeFillNotes(result)
			targetActive := int(float64(countActive(result)) * 0.8)
			deactivateSteps(result, countActive(result)-targetActive)
		case "density_up":
			count := int(float64(3) * ev.Intensity)
			activateSteps(result, count)
		case "density_down":
			count := int(float64(3) * ev.Intensity)
			deactivateSteps(result, count)
		}
	}

	// REFACTOR.md point 5: Bar-to-bar variation — mutate 1-2 steps every 4 bars
	if bar > 0 && bar%4 == 0 {
		mutateMotif(result, 1+(bar/4)%2) // mutate 1-2 steps
	}

	return result
}

// addFillNotes adds ghost notes on off-beat positions for build sections.
func addFillNotes(steps []schema.StepSpec, count int) {
	if count <= 0 {
		return
	}
	added := 0
	// Add ghost notes on off-beat positions (odd indices)
	for i := 1; i < len(steps) && added < count; i += 2 {
		if !steps[i].Active && steps[i].Note != "" {
			steps[i].Active = true
			steps[i].Ghost = true
			added++
		}
	}
}

// removeFillNotes removes ghost notes added during build.
func removeFillNotes(steps []schema.StepSpec) {
	for i := range steps {
		if steps[i].Ghost && i%2 == 1 {
			steps[i].Active = false
		}
	}
}

// mutateMotif introduces small variations to the motif by changing 1-2 steps.
func mutateMotif(steps []schema.StepSpec, count int) {
	if count <= 0 {
		return
	}
	mutated := 0
	// Mutate non-downbeat active steps
	for i := 2; i < len(steps) && mutated < count; i += 4 {
		if steps[i].Active && !steps[i].Accent {
			// Toggle activity or add/remove ghost flag
			if steps[i].Ghost {
				steps[i].Ghost = false
			} else {
				steps[i].Ghost = true
			}
			mutated++
		}
	}
}

func applyOctaveShift(steps []schema.StepSpec, delta int, fraction float64) {
	active := activeIndices(steps)
	count := int(float64(len(active)) * fraction)
	if count < 1 {
		count = 1
	}
	for i := 0; i < count && i < len(active); i++ {
		idx := active[i]
		steps[idx].Note = shiftOctave(steps[idx].Note, delta)
	}
}

func deactivateSteps(steps []schema.StepSpec, count int) {
	if count <= 0 {
		return
	}
	deactivated := 0
	for i := len(steps) - 1; i >= 0 && deactivated < count; i -= 2 {
		if i == 0 {
			continue
		}
		if steps[i].Active {
			steps[i].Active = false
			deactivated++
		}
	}
	for i := len(steps) - 2; i >= 0 && deactivated < count; i -= 2 {
		if i == 0 {
			continue
		}
		if steps[i].Active {
			steps[i].Active = false
			deactivated++
		}
	}
}

func activateSteps(steps []schema.StepSpec, count int) {
	if count <= 0 {
		return
	}
	activated := 0
	for i := 0; i < len(steps) && activated < count; i += 2 {
		if !steps[i].Active {
			steps[i].Active = true
			activated++
		}
	}
	for i := 1; i < len(steps) && activated < count; i += 2 {
		if !steps[i].Active {
			steps[i].Active = true
			activated++
		}
	}
}

func countActive(steps []schema.StepSpec) int {
	n := 0
	for _, s := range steps {
		if s.Active {
			n++
		}
	}
	return n
}

func activeIndices(steps []schema.StepSpec) []int {
	var indices []int
	for i, s := range steps {
		if s.Active && s.Note != "" {
			indices = append(indices, i)
		}
	}
	return indices
}

func shiftOctave(note string, delta int) string {
	if len(note) < 2 {
		return note
	}
	last := note[len(note)-1]
	if last < '0' || last > '9' {
		return note
	}
	newOctave := int(last-'0') + delta
	if newOctave < 0 || newOctave > 9 {
		return note
	}
	return note[:len(note)-1] + string(rune('0'+newOctave))
}
