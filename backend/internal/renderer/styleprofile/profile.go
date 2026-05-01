package styleprofile

// StyleProfile is a deterministic rendering profile controlling timing, velocity, gate, and automation.
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

// TimingProfile controls per-step timing offsets and downbeat rigidity.
type TimingProfile struct {
	DownbeatRigid  bool
	OffbeatOffset  []int
	HumanizedDelay bool
}

// VelocityProfile defines accent grid, ghost velocity, and dynamic curve scaling.
type VelocityProfile struct {
	AccentGrid    []int
	GhostVelocity int
	AccentBoost   int
	MaxVelocity   int
	DynamicCurve  string // "flat" (default), "crescendo", "arch"
}

// GateProfile controls note duration ratios for different articulations.
type GateProfile struct {
	NormalGate   float64
	AccentGate   float64
	GhostGate    float64
	SlideGate    float64
	StaccatoGate float64
	LegatoGate   float64
}

// PortamentoProfile configures CC5/CC65 portamento behavior for slide notes.
type PortamentoProfile struct {
	CC5Value int
	UseCC65  bool
}

// FilterSweepProfile configures CC74 filter cutoff automation curve and range.
type FilterSweepProfile struct {
	Curve       string
	Resolution  int
	Jitter      int
	KickbackEnd bool
	Range       map[string][2]int
}

// ModWheelProfile configures CC1 modulation wheel automation.
type ModWheelProfile struct {
	Enabled    bool
	Resolution int
	Range      [2]int
	Curve      string
}

// NoteRangeConstraint defines valid MIDI note bounds for a pattern type.
type NoteRangeConstraint struct {
	MinMIDI int
	MaxMIDI int
}

// DensityConstraint defines how many steps must be active/silent in a motif.
type DensityConstraint struct {
	MinActiveSteps int
	MaxActiveSteps int
	MinSilence     int
}
