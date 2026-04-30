package renderer

import "github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"

// stepFlags groups per-step articulation flags to keep function signatures compact.
type stepFlags struct {
	Accent, Ghost, Slide, Legato, Staccato bool
}

func resolveGateTicksExt(flags stepFlags, profile *styleprofile.StyleProfile, tick, nextActiveTick int64) int64 {
	if flags.Slide && nextActiveTick > tick && nextActiveTick > 0 {
		return nextActiveTick - tick + 5
	}

	var ratio float64

	switch {
	case flags.Slide && profile.Gate.SlideGate > 0:
		ratio = profile.Gate.SlideGate
	case flags.Legato && profile.Gate.LegatoGate > 0:
		ratio = profile.Gate.LegatoGate
	case flags.Staccato && profile.Gate.StaccatoGate > 0:
		ratio = profile.Gate.StaccatoGate
	case flags.Ghost:
		ratio = profile.Gate.GhostGate
	case flags.Accent:
		ratio = profile.Gate.AccentGate
	default:
		ratio = profile.Gate.NormalGate
	}

	if ratio <= 0 {
		ratio = 0.72
	}

	return int64(float64(ticksPerStep) * ratio)
}
