package renderer

import "github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"

const ticksPerStep = 120 // 480 ticks/beat ÷ 4 steps/beat

func resolveTick(bar, stepIdx int, profile *styleprofile.StyleProfile) int64 {
	baseTick := int64(bar*16+stepIdx) * ticksPerStep

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
