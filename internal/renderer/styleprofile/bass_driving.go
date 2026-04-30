package styleprofile

var BassDriving = StyleProfile{
	Name:        "bass_techno_driving",
	PatternType: "bassline",
	Timing: TimingProfile{
		DownbeatRigid: true,
		OffbeatOffset: []int{0, +3, -2, +4, 0, +3, -1, +5, 0, +2, -2, +4, 0, +3, -1, +6},
	},
	Velocity: VelocityProfile{
		AccentGrid:    []int{120, 82, 95, 78, 115, 80, 92, 75, 118, 82, 95, 78, 112, 80, 90, 82},
		GhostVelocity: 38,
		AccentBoost:   10,
		MaxVelocity:   120,
		DynamicCurve:  "plateau",
	},
	Gate: GateProfile{
		NormalGate: 0.60,
		AccentGate: 0.78,
		GhostGate:  0.22,
		SlideGate:  1.10,
	},
	Portamento: PortamentoProfile{
		CC5Value: 45,
		UseCC65:  true,
	},
	FilterSweep: FilterSweepProfile{
		Curve:       "exponential",
		Resolution:  16,
		Jitter:      3,
		KickbackEnd: true,
		Range: map[string][2]int{
			"subtle":   {50, 80},
			"medium":   {35, 100},
			"dramatic": {20, 115},
		},
	},
	NoteRange: NoteRangeConstraint{MinMIDI: 33, MaxMIDI: 50},
	Density:   DensityConstraint{MinActiveSteps: 10, MaxActiveSteps: 14},
}
