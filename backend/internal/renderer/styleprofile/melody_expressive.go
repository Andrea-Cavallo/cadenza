package styleprofile

var MelodyExpressive = StyleProfile{
	Name:        "melody_expressive",
	PatternType: "melody",
	Timing: TimingProfile{
		DownbeatRigid:  false,
		OffbeatOffset:  []int{0, -2, +3, -1, 0, +2, -3, +1, 0, -2, +3, 0, +2, -1, +3, -2},
		HumanizedDelay: true,
	},
	Velocity: VelocityProfile{
		AccentGrid:    []int{105, 92, 88, 95, 110, 85, 90, 98, 108, 93, 87, 100, 105, 90, 95, 88},
		GhostVelocity: 50,
		AccentBoost:   10,
		MaxVelocity:   120,
		DynamicCurve:  "crescendo",
	},
	Gate: GateProfile{
		NormalGate: 0.65,
		AccentGate: 0.85,
		GhostGate:  0.30,
		LegatoGate: 0.98,
	},
	ModWheel: ModWheelProfile{
		Enabled:    true,
		Resolution: 16,
		Range:      [2]int{0, 80},
		Curve:      "phrase_based",
	},
	NoteRange: NoteRangeConstraint{MinMIDI: 60, MaxMIDI: 96},
	Density:   DensityConstraint{MinActiveSteps: 4, MaxActiveSteps: 10, MinSilence: 3},
}
