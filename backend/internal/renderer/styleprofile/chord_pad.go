package styleprofile

var ChordPadGroove = StyleProfile{
	Name:        "chord_pad_groove",
	PatternType: "chord_pad",
	Timing: TimingProfile{
		DownbeatRigid: true,
		OffbeatOffset: []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	Velocity: VelocityProfile{
		AccentGrid:    []int{78, 60, 62, 58, 72, 60, 62, 58, 78, 60, 62, 58, 72, 60, 62, 58},
		GhostVelocity: 38,
		AccentBoost:   5,
		MaxVelocity:   85,
		DynamicCurve:  "flat",
	},
	Gate: GateProfile{
		NormalGate: 0.92,
		AccentGate: 0.95,
		GhostGate:  0.50,
		SlideGate:  1.00,
		LegatoGate: 0.98,
	},
	Portamento: PortamentoProfile{CC5Value: 0, UseCC65: false},
	FilterSweep: FilterSweepProfile{
		Curve:      "flat",
		Resolution: 4,
		Range: map[string][2]int{
			"subtle": {70, 85},
			"medium": {60, 90},
		},
	},
	NoteRange: NoteRangeConstraint{MinMIDI: 48, MaxMIDI: 79},
	Density:   DensityConstraint{MinActiveSteps: 2, MaxActiveSteps: 6},
}

var ChordPadRolling = StyleProfile{
	Name:        "chord_pad_rolling",
	PatternType: "chord_pad",
	Timing:      TimingProfile{DownbeatRigid: true},
	Velocity: VelocityProfile{
		AccentGrid:    []int{60, 45, 45, 45, 45, 45, 45, 45, 60, 45, 45, 45, 45, 45, 45, 45},
		GhostVelocity: 32,
		AccentBoost:   3,
		MaxVelocity:   65,
		DynamicCurve:  "flat",
	},
	Gate: GateProfile{
		NormalGate: 0.96,
		AccentGate: 0.98,
		GhostGate:  0.60,
		LegatoGate: 0.99,
	},
	Portamento:  PortamentoProfile{CC5Value: 0, UseCC65: false},
	FilterSweep: FilterSweepProfile{Curve: "flat", Resolution: 4, Range: map[string][2]int{"subtle": {72, 82}}},
	NoteRange:   NoteRangeConstraint{MinMIDI: 48, MaxMIDI: 79},
	Density:     DensityConstraint{MinActiveSteps: 2, MaxActiveSteps: 2},
}

var ChordPadSub = StyleProfile{
	Name:        "chord_pad_sub",
	PatternType: "chord_pad",
	Timing:      TimingProfile{DownbeatRigid: true},
	Velocity: VelocityProfile{
		AccentGrid:    []int{78, 62, 65, 60, 75, 62, 65, 60, 78, 62, 65, 60, 75, 62, 65, 60},
		GhostVelocity: 40,
		AccentBoost:   5,
		MaxVelocity:   85,
		DynamicCurve:  "flat",
	},
	Gate: GateProfile{
		NormalGate: 0.97,
		AccentGate: 0.99,
		GhostGate:  0.55,
		LegatoGate: 1.00,
	},
	Portamento:  PortamentoProfile{CC5Value: 0, UseCC65: false},
	FilterSweep: FilterSweepProfile{Curve: "s_curve", Resolution: 8, Range: map[string][2]int{"subtle": {60, 80}, "medium": {50, 90}}},
	NoteRange:   NoteRangeConstraint{MinMIDI: 36, MaxMIDI: 72},
	Density:     DensityConstraint{MinActiveSteps: 4, MaxActiveSteps: 6},
}
