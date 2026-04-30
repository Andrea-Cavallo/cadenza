package renderer

import "github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"

func resolveGateTicksExt(stepIdx int, accent, ghost, slide, legato, staccato bool, profile *styleprofile.StyleProfile, tick, nextActiveTick int64) int64 {
	if slide && nextActiveTick > tick && nextActiveTick > 0 {
		return nextActiveTick - tick + 5
	}

	var ratio float64

	switch {
	case slide && profile.Gate.SlideGate > 0:
		ratio = profile.Gate.SlideGate
	case legato && profile.Gate.LegatoGate > 0:
		ratio = profile.Gate.LegatoGate
	case staccato && profile.Gate.StaccatoGate > 0:
		ratio = profile.Gate.StaccatoGate
	case ghost:
		ratio = profile.Gate.GhostGate
	case accent:
		ratio = profile.Gate.AccentGate
	default:
		ratio = profile.Gate.NormalGate
	}

	if ratio <= 0 {
		ratio = 0.72
	}

	return int64(float64(ticksPerStep) * ratio)
}
