package styleprofile

var BassRolling = StyleProfile{
	Name:        "bass_rolling",
	PatternType: "bassline",
	Timing: TimingProfile{
		DownbeatRigid: true,
		OffbeatOffset: []int{0, +1, -1, +2, 0, +1, -1, +2, 0, +1, -1, +2, 0, +1, -1, +2},
	},
	Velocity: VelocityProfile{
		AccentGrid:    []int{112, 78, 82, 75, 108, 80, 84, 76, 112, 78, 82, 75, 108, 80, 84, 76},
		GhostVelocity: 44,
		AccentBoost:   10,
		MaxVelocity:   115,
		DynamicCurve:  "crescendo",
	},
	Gate: GateProfile{
		NormalGate:   0.65,
		AccentGate:   0.80,
		GhostGate:    0.22,
		SlideGate:    1.05,
		StaccatoGate: 0.18,
		LegatoGate:   0.90,
	},
	Portamento: PortamentoProfile{
		CC5Value: 50,
		UseCC65:  true,
	},
	FilterSweep: FilterSweepProfile{
		Curve:       "exponential",
		Resolution:  16,
		Jitter:      3,
		KickbackEnd: true,
		Range: map[string][2]int{
			"subtle":   {45, 90},
			"medium":   {30, 105},
			"dramatic": {20, 118},
		},
	},
	NoteRange: NoteRangeConstraint{MinMIDI: 33, MaxMIDI: 55},
	Density:   DensityConstraint{MinActiveSteps: 14, MaxActiveSteps: 16},
}
