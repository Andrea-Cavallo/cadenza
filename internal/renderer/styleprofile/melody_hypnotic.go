package styleprofile

var MelodyHypnotic = StyleProfile{
	Name:        "melody_hypnotic",
	PatternType: "melody",
	Timing: TimingProfile{
		DownbeatRigid:  true,
		OffbeatOffset:  []int{0, +1, -1, 0, 0, +2, -1, 0, 0, +1, 0, -1, 0, +1, -2, 0},
		HumanizedDelay: true,
	},
	Velocity: VelocityProfile{
		AccentGrid:    []int{100, 88, 85, 88, 95, 85, 88, 85, 98, 88, 85, 88, 95, 85, 88, 85},
		GhostVelocity: 52,
		AccentBoost:   5,
		MaxVelocity:   110,
		DynamicCurve:  "crescendo",
	},
	Gate: GateProfile{
		NormalGate: 0.90,
		AccentGate: 0.95,
		GhostGate:  0.45,
		LegatoGate: 0.98,
	},
	ModWheel: ModWheelProfile{
		Enabled:    true,
		Resolution: 8,
		Range:      [2]int{10, 70},
		Curve:      "phrase_based",
	},
	NoteRange: NoteRangeConstraint{MinMIDI: 60, MaxMIDI: 84},
	Density:   DensityConstraint{MinActiveSteps: 3, MaxActiveSteps: 8, MinSilence: 4},
}
