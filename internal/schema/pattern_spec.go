package schema

// PatternSpec is the LLM-generated musical pattern specification validated before rendering.
type PatternSpec struct {
	SpecVersion   string           `json:"spec_version"`
	PatternType   string           `json:"pattern_type"`
	Meta          PatternMeta      `json:"meta"`
	Theory        TheorySpec       `json:"theory"`
	StyleProfile  string           `json:"style_profile"`
	Motif         MotifSpec        `json:"motif"`
	Evolution     []EvolutionStep  `json:"evolution"`
	Automation    AutomationIntent `json:"automation"`
	VariationSeed string           `json:"variation_seed"`
}

// PatternMeta holds descriptive metadata for a pattern (BPM, key, bar count).
type PatternMeta struct {
	Name        string  `json:"name"`
	BPM         float64 `json:"bpm"`
	Key         string  `json:"key"`
	Bars        int     `json:"bars"`
	Description string  `json:"description"`
}

// TheorySpec declares the musical theory constraints (key, mode, scale, octave range).
type TheorySpec struct {
	Key         string `json:"key"`
	Mode        string `json:"mode"`
	Scale       string `json:"scale"`
	OctaveRange [2]int `json:"octave_range"`
}

// MotifSpec defines the 16-step motif pattern with per-step note data.
type MotifSpec struct {
	Length int        `json:"length"`
	Steps  []StepSpec `json:"steps"`
}

// StepSpec represents a single step in the motif with articulation flags.
type StepSpec struct {
	Active   bool   `json:"active"`
	Note     string `json:"note,omitempty"`
	Accent   bool   `json:"accent,omitempty"`
	Slide    bool   `json:"slide,omitempty"`
	Ghost    bool   `json:"ghost,omitempty"`
	Legato   bool   `json:"legato,omitempty"`
	Staccato bool   `json:"staccato,omitempty"`
}

// EvolutionStep defines how the pattern evolves across a bar range.
type EvolutionStep struct {
	FromBar   int     `json:"from_bar"`
	ToBar     int     `json:"to_bar"`
	Action    string  `json:"action"`
	Intensity float64 `json:"intensity"`
}

// AutomationIntent declares optional CC automation layers (filter, mod wheel, pitch bend).
type AutomationIntent struct {
	FilterSweep *FilterSweepIntent `json:"filter_sweep,omitempty"`
	ModWheel    *ModWheelIntent    `json:"mod_wheel,omitempty"`
	PitchBend   *PitchBendIntent   `json:"pitch_bend,omitempty"`
}

// FilterSweepIntent configures CC74 filter cutoff automation style.
type FilterSweepIntent struct {
	Style string `json:"style"` // "subtle", "medium", "dramatic"
}

// ModWheelIntent configures CC1 modulation wheel automation style.
type ModWheelIntent struct {
	Style string `json:"style"`
}

// PitchBendIntent configures pitch bend automation style.
type PitchBendIntent struct {
	Style string `json:"style"`
}
