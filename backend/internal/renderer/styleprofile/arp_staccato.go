package styleprofile

var ArpStaccato = StyleProfile{
	Name:        "arp_staccato",
	PatternType: "arpeggio",
	Timing: TimingProfile{
		DownbeatRigid: true,
		OffbeatOffset: []int{0, -1, +1, -1, 0, +1, -1, +1, 0, -1, +1, 0, 0, +1, -1, +1},
	},
	Velocity: VelocityProfile{
		AccentGrid:    []int{95, 90, 92, 88, 95, 88, 90, 85, 95, 90, 92, 88, 95, 88, 90, 85},
		GhostVelocity: 50,
		AccentBoost:   5,
		MaxVelocity:   110,
		DynamicCurve:  "flat",
	},
	Gate: GateProfile{
		NormalGate:   0.20,
		AccentGate:   0.25,
		GhostGate:    0.15,
		StaccatoGate: 0.12,
		LegatoGate:   0.30,
	},
	FilterSweep: FilterSweepProfile{
		Curve:       "linear",
		Resolution:  16,
		Jitter:      2,
		KickbackEnd: false,
		Range: map[string][2]int{
			"subtle":   {70, 100},
			"medium":   {60, 110},
			"dramatic": {50, 120},
		},
	},
	NoteRange: NoteRangeConstraint{MinMIDI: 48, MaxMIDI: 84},
	Density:   DensityConstraint{MinActiveSteps: 10, MaxActiveSteps: 16},
}
