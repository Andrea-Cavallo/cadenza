package renderer

import "github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"

const ticksPerStep = 120 // 480 ticks/beat ÷ 4 steps/beat

// resolveTick computes the absolute MIDI tick for a step within a renderer bar.
// stepsPerBar is len(barSteps), allowing motifs of any length (16 or 64 steps).
func resolveTick(bar, stepIdx, stepsPerBar int, profile *styleprofile.StyleProfile) int64 {
	baseTick := int64(bar*stepsPerBar+stepIdx) * ticksPerStep

	if stepIdx == 0 && profile.Timing.DownbeatRigid {
		return baseTick
	}

	if len(profile.Timing.OffbeatOffset) == 0 {
		return baseTick
	}

	offsetIdx := stepIdx % len(profile.Timing.OffbeatOffset)
	offset := int64(profile.Timing.OffbeatOffset[offsetIdx])

	tick := baseTick + offset
	if tick < 0 {
		tick = 0
	}
	return tick
}
