package schema

type PatternSpec struct {
	SpecVersion   string          `json:"spec_version"`
	PatternType   string          `json:"pattern_type"`
	Meta          PatternMeta     `json:"meta"`
	Theory        TheorySpec      `json:"theory"`
	StyleProfile  string          `json:"style_profile"`
	Motif         MotifSpec       `json:"motif"`
	Evolution     []EvolutionStep `json:"evolution"`
	Automation    AutomationIntent `json:"automation"`
	VariationSeed string          `json:"variation_seed"`
}

type PatternMeta struct {
	Name        string  `json:"name"`
	BPM         float64 `json:"bpm"`
	Key         string  `json:"key"`
	Bars        int     `json:"bars"`
	Description string  `json:"description"`
}

type TheorySpec struct {
	Key         string `json:"key"`
	Mode        string `json:"mode"`
	Scale       string `json:"scale"`
	OctaveRange [2]int `json:"octave_range"`
}

type MotifSpec struct {
	Length int        `json:"length"`
	Steps  []StepSpec `json:"steps"`
}

type StepSpec struct {
	Active   bool   `json:"active"`
	Note     string `json:"note,omitempty"`
	Accent   bool   `json:"accent,omitempty"`
	Slide    bool   `json:"slide,omitempty"`
	Ghost    bool   `json:"ghost,omitempty"`
	Legato   bool   `json:"legato,omitempty"`
	Staccato bool   `json:"staccato,omitempty"`
}

type EvolutionStep struct {
	FromBar   int     `json:"from_bar"`
	ToBar     int     `json:"to_bar"`
	Action    string  `json:"action"`
	Intensity float64 `json:"intensity"`
}

type AutomationIntent struct {
	FilterSweep *FilterSweepIntent `json:"filter_sweep,omitempty"`
	ModWheel    *ModWheelIntent    `json:"mod_wheel,omitempty"`
	PitchBend   *PitchBendIntent   `json:"pitch_bend,omitempty"`
}

type FilterSweepIntent struct {
	Style string `json:"style"` // "subtle", "medium", "dramatic"
}

type ModWheelIntent struct {
	Style string `json:"style"`
}

type PitchBendIntent struct {
	Style string `json:"style"`
}
