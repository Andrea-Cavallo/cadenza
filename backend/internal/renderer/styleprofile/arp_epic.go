package styleprofile

var ArpEpic = StyleProfile{
	Name:        "arp_epic",
	PatternType: "arpeggio",
	Timing: TimingProfile{
		DownbeatRigid: true,
		OffbeatOffset: []int{0, 0, +1, 0, 0, 0, +1, 0, 0, 0, +1, 0, 0, 0, +1, 0},
	},
	Velocity: VelocityProfile{
		AccentGrid:    []int{100, 75, 80, 72, 95, 72, 78, 70, 98, 74, 82, 73, 100, 76, 85, 74},
		GhostVelocity: 55,
		AccentBoost:   8,
		MaxVelocity:   115,
		DynamicCurve:  "tension",
	},
	Gate: GateProfile{
		NormalGate:   0.55,
		AccentGate:   0.65,
		GhostGate:    0.35,
		StaccatoGate: 0.28,
		LegatoGate:   0.82,
	},
	FilterSweep: FilterSweepProfile{
		Curve:       "s_curve",
		Resolution:  32,
		Jitter:      1,
		KickbackEnd: true,
		Range: map[string][2]int{
			"subtle":   {50, 95},
			"medium":   {30, 110},
			"dramatic": {20, 120},
		},
	},
	NoteRange: NoteRangeConstraint{MinMIDI: 48, MaxMIDI: 84},
	Density:   DensityConstraint{MinActiveSteps: 8, MaxActiveSteps: 16},
}
