package styleprofile

var ArpFlowing = StyleProfile{
	Name:        "arp_flowing",
	PatternType: "arpeggio",
	Timing: TimingProfile{
		DownbeatRigid: false,
		OffbeatOffset: []int{0, +1, -1, +2, 0, +1, -1, +2, 0, +1, -1, +2, 0, +1, -1, +2},
	},
	Velocity: VelocityProfile{
		AccentGrid:    []int{95, 88, 92, 85, 98, 90, 93, 87, 96, 89, 91, 86, 94, 88, 90, 85},
		GhostVelocity: 60,
		AccentBoost:   5,
		MaxVelocity:   110,
		DynamicCurve:  "arch",
	},
	Gate: GateProfile{
		NormalGate:   0.48,
		AccentGate:   0.60,
		GhostGate:    0.25,
		StaccatoGate: 0.28,
		LegatoGate:   0.78,
	},
	FilterSweep: FilterSweepProfile{
		Curve:      "exponential",
		Resolution: 32,
		Jitter:     1,
		Range: map[string][2]int{
			"subtle":   {65, 90},
			"medium":   {50, 105},
			"dramatic": {35, 115},
		},
	},
	NoteRange: NoteRangeConstraint{MinMIDI: 48, MaxMIDI: 84},
	Density:   DensityConstraint{MinActiveSteps: 12, MaxActiveSteps: 16},
}
