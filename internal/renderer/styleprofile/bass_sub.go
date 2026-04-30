package styleprofile

var BassSub = StyleProfile{
	Name:        "bass_sub",
	PatternType: "bassline",
	Timing: TimingProfile{
		DownbeatRigid: true,
		OffbeatOffset: []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	Velocity: VelocityProfile{
		AccentGrid:    []int{90, 85, 88, 82, 90, 85, 88, 82, 90, 85, 88, 82, 90, 85, 88, 82},
		GhostVelocity: 40,
		AccentBoost:   3,
		MaxVelocity:   95,
		DynamicCurve:  "flat",
	},
	Gate: GateProfile{
		NormalGate: 0.95,
		AccentGate: 0.98,
		GhostGate:  0.30,
		SlideGate:  1.00,
	},
	Portamento: PortamentoProfile{
		CC5Value: 0,
		UseCC65:  false,
	},
	FilterSweep: FilterSweepProfile{
		Curve:       "flat",
		Resolution:  4,
		Jitter:      0,
		KickbackEnd: false,
		Range: map[string][2]int{
			"subtle":   {60, 70},
			"medium":   {55, 75},
			"dramatic": {50, 80},
		},
	},
	NoteRange: NoteRangeConstraint{MinMIDI: 28, MaxMIDI: 48},
	Density:   DensityConstraint{MinActiveSteps: 4, MaxActiveSteps: 8},
}
