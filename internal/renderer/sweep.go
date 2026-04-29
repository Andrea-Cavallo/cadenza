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

func filterSweepEvents(bar, totalBars int, sweepStyle, variationSeed string, profile *styleprofile.StyleProfile, evolutions []schema.EvolutionStep) []midi.MIDIEvent {
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

	barStartTick := int64(bar * 16 * ticksPerStep)
	stepTicks := int64(16*ticksPerStep) / int64(resolution)

	events := make([]midi.MIDIEvent, 0, resolution)
	for step := 0; step < resolution; step++ {
		progress := float64(bar*resolution+step) / float64(totalBars*resolution)

		var value float64
		switch profile.FilterSweep.Curve {
		case "s_curve":
			value = minVal + (maxVal-minVal)*sCurve(progress)
		case "exponential":
			value = minVal + (maxVal-minVal)*math.Pow(progress, 2.0)
		default:
			value = minVal + (maxVal-minVal)*progress
		}

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
	fmt.Fprintf(h, "%s:%d:%d", seed, bar, step)
	v := int(h.Sum32() % uint32(maxJitter*2+1))
	return v - maxJitter
}

func sCurve(t float64) float64 {
	return t * t * (3.0 - 2.0*t)
}
