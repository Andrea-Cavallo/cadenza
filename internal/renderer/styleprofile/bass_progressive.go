package styleprofile

var BassProgressive = StyleProfile{
	Name:        "bass_progressive",
	PatternType: "bassline",
	Timing: TimingProfile{
		DownbeatRigid: true,
		OffbeatOffset: []int{0, +2, -1, +3, 0, +2, -1, +4, 0, +2, -1, +3, 0, +2, -1, +5},
	},
	Velocity: VelocityProfile{
		AccentGrid:    []int{118, 75, 85, 70, 110, 75, 88, 72, 115, 78, 85, 70, 108, 75, 90, 80},
		GhostVelocity: 42,
		AccentBoost:   8,
		MaxVelocity:   120,
		DynamicCurve:  "crescendo",
	},
	Gate: GateProfile{
		NormalGate: 0.72,
		AccentGate: 0.88,
		GhostGate:  0.28,
		SlideGate:  1.05,
	},
	Portamento: PortamentoProfile{
		CC5Value: 35,
		UseCC65:  true,
	},
	FilterSweep: FilterSweepProfile{
		Curve:       "s_curve",
		Resolution:  16,
		Jitter:      2,
		KickbackEnd: true,
		Range: map[string][2]int{
			"subtle":   {55, 85},
			"medium":   {40, 100},
			"dramatic": {25, 115},
		},
	},
	NoteRange: NoteRangeConstraint{MinMIDI: 33, MaxMIDI: 55},
	Density:   DensityConstraint{MinActiveSteps: 8, MaxActiveSteps: 13},
}
