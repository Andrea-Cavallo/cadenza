package styleprofile

var LeadStab = StyleProfile{
	Name:        "lead_stab",
	PatternType: "lead_stab",
	Timing: TimingProfile{
		DownbeatRigid: true,
		OffbeatOffset: []int{0, +1, -1, +2, 0, +1, -1, +3, 0, +1, -1, +2, 0, +1, -1, +3},
	},
	Velocity: VelocityProfile{
		AccentGrid:    []int{100, 72, 78, 68, 95, 70, 75, 65, 102, 72, 78, 68, 95, 70, 75, 65},
		GhostVelocity: 0, // no ghost on lead stab
		AccentBoost:   8,
		MaxVelocity:   110,
		DynamicCurve:  "crescendo",
	},
	Gate: GateProfile{
		NormalGate:   0.20,
		AccentGate:   0.25,
		GhostGate:    0.12,
		StaccatoGate: 0.15,
		SlideGate:    0.25,
	},
	Portamento:  PortamentoProfile{CC5Value: 0, UseCC65: false},
	FilterSweep: FilterSweepProfile{Curve: "flat", Resolution: 4, Range: map[string][2]int{"subtle": {65, 85}}},
	NoteRange:   NoteRangeConstraint{MinMIDI: 60, MaxMIDI: 96},
	Density:     DensityConstraint{MinActiveSteps: 1, MaxActiveSteps: 8},
}
