package renderer

import "github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"

func resolveVelocity(stepIdx int, accent, ghost bool, profile *styleprofile.StyleProfile, scale float64) uint8 {
	if ghost {
		v := int(float64(profile.Velocity.GhostVelocity) * scale)
		if v < 1 {
			v = 1
		}
		if v > 60 {
			v = 60
		}
		return uint8(v)
	}

	gridIdx := stepIdx % len(profile.Velocity.AccentGrid)
	vel := profile.Velocity.AccentGrid[gridIdx]

	if accent {
		vel += profile.Velocity.AccentBoost
	}

	vel = int(float64(vel) * scale)

	if vel > profile.Velocity.MaxVelocity {
		vel = profile.Velocity.MaxVelocity
	}
	if vel < 1 {
		vel = 1
	}
	return uint8(vel)
}
