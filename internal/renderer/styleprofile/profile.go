package styleprofile

type StyleProfile struct {
	Name        string
	PatternType string // "bassline", "arpeggio", "melody"
	Timing      TimingProfile
	Velocity    VelocityProfile
	Gate        GateProfile
	Portamento  PortamentoProfile
	FilterSweep FilterSweepProfile
	ModWheel    ModWheelProfile
	NoteRange   NoteRangeConstraint
	Density     DensityConstraint
}

type TimingProfile struct {
	DownbeatRigid  bool
	OffbeatOffset  []int
	HumanizedDelay bool
}

type VelocityProfile struct {
	AccentGrid    []int
	GhostVelocity int
	AccentBoost   int
	MaxVelocity   int
	DynamicCurve  string // "flat" (default), "crescendo", "arch"
}

type GateProfile struct {
	NormalGate   float64
	AccentGate   float64
	GhostGate    float64
	SlideGate    float64
	StaccatoGate float64
	LegatoGate   float64
}

type PortamentoProfile struct {
	CC5Value int
	UseCC65  bool
}

type FilterSweepProfile struct {
	Curve       string
	Resolution  int
	Jitter      int
	KickbackEnd bool
	Range       map[string][2]int
}

type ModWheelProfile struct {
	Enabled    bool
	Resolution int
	Range      [2]int
	Curve      string
}

type NoteRangeConstraint struct {
	MinMIDI int
	MaxMIDI int
}

type DensityConstraint struct {
	MinActiveSteps int
	MaxActiveSteps int
	MinSilence     int
}
