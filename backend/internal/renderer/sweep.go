package renderer

import (
	"fmt"
	"hash/fnv"
	"math"

	"github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
)

const ccFilterCutoff = uint8(74)

// REFACTOR.md point 5: Pattern-type-specific phase offsets to prevent simultaneous sweep peaks
var sweepPhaseOffsets = map[string]float64{
	"bassline": 0.0,
	"arpeggio": 0.25,
	"melody":   0.5,
}

// computeSweepProgress computes the normalised [0,1] progress within the current cycle.
func computeSweepProgress(bar, step, cycleBars, resolution int, phaseOffset float64) float64 {
	cycleBar := bar % cycleBars
	base := (float64(cycleBar*resolution+step) + phaseOffset*float64(cycleBars*resolution)) / float64(cycleBars*resolution)
	p := math.Mod(base, 1.0)
	if p < 0 {
		p += 1.0
	}
	return p
}

// computeSweepValue computes the raw CC74 value for the given progress and sweep style.
func computeSweepValue(sweepStyle string, progress float64, minVal, maxVal float64, cycleCount int, profile *styleprofile.StyleProfile) float64 {
	if sweepStyle == "dramatic" {
		if cycleCount%2 == 0 {
			// First cycle: up-down
			if progress < 0.5 {
				return minVal + (maxVal-minVal)*applyCurve(progress*2, profile.FilterSweep.Curve)
			}
			return maxVal - (maxVal-minVal)*applyCurve((progress-0.5)*2, profile.FilterSweep.Curve)
		}
		// Second cycle: up to peak
		return minVal + (maxVal-minVal)*applyCurve(progress, profile.FilterSweep.Curve)
	}
	return minVal + (maxVal-minVal)*applyCurve(progress, profile.FilterSweep.Curve)
}

func filterSweepEvents(bar, totalBars int, sweepStyle, variationSeed, patternType string, profile *styleprofile.StyleProfile, evolutions []schema.EvolutionStep) []midi.MIDIEvent {
	if profile.FilterSweep.Resolution == 0 {
		return nil
	}

	sweepRange, ok := profile.FilterSweep.Range[sweepStyle]
	if !ok {
		sweepRange = profile.FilterSweep.Range["medium"]
	}

	baseMin := float64(sweepRange[0])
	baseMax := float64(sweepRange[1])

	intensity := evolutionIntensityForBar(bar, evolutions)
	spread := (baseMax - baseMin) * intensity
	minVal := baseMax - spread
	maxVal := baseMax

	resolution := profile.FilterSweep.Resolution
	jitter := profile.FilterSweep.Jitter

	// Multi-cycle filter sweep: 4-bar cycle for all styles
	cycleBars := 4

	// REFACTOR.md point 5: Apply pattern-type-specific phase offset
	phaseOffset := sweepPhaseOffsets[patternType]
	if phaseOffset == 0 && patternType != "bassline" {
		// Default fallback if pattern type not in map
		phaseOffset = 0.0
	}

	barStartTick := int64(bar * 16 * ticksPerStep)
	stepTicks := int64(16*ticksPerStep) / int64(resolution)

	events := make([]midi.MIDIEvent, 0, resolution)
	for step := 0; step < resolution; step++ {
		progress := computeSweepProgress(bar, step, cycleBars, resolution, phaseOffset)
		cycleCount := bar / cycleBars
		value := computeSweepValue(sweepStyle, progress, minVal, maxVal, cycleCount, profile)

		if jitter > 0 {
			value += float64(deterministicJitter(variationSeed, bar, step, jitter))
		}

		value = math.Max(0, math.Min(127, value))

		if profile.FilterSweep.KickbackEnd && bar == totalBars-1 && step == resolution-1 {
			value = maxVal - 10
			if value > 127 {
				value = 127
			}
		}

		events = append(events, midi.MIDIEvent{
			Type:       midi.ControlChange,
			Tick:       barStartTick + int64(step)*stepTicks,
			Channel:    0,
			Controller: ccFilterCutoff,
			Value:      uint8(math.Round(value)),
		})
	}

	return events
}

// applyCurve applies the specified curve function to a progress value [0, 1].
func applyCurve(progress float64, curve string) float64 {
	switch curve {
	case "s_curve":
		return sCurve(progress)
	case "exponential":
		return math.Pow(progress, 2.0)
	default:
		return progress
	}
}

func evolutionIntensityForBar(bar int, evolutions []schema.EvolutionStep) float64 {
	barNum := bar + 1
	for _, ev := range evolutions {
		if barNum >= ev.FromBar && barNum <= ev.ToBar {
			return ev.Intensity
		}
	}
	return 0.5
}

func deterministicJitter(seed string, bar, step, maxJitter int) int {
	h := fnv.New32a()
	_, _ = fmt.Fprintf(h, "%s:%d:%d", seed, bar, step)
	v := int(h.Sum32() % uint32(maxJitter*2+1))
	return v - maxJitter
}

func sCurve(t float64) float64 {
	return t * t * (3.0 - 2.0*t)
}
