package examples

// RawNote is a MIDI note quantized to the 16th-note grid.
// Step is the absolute step index (not wrapped to 0-15).
type RawNote struct {
	Step          int   // absolute step (0, 16, 32, ... for multi-bar files)
	MIDIPitch     int   // absolute MIDI pitch 0-127
	DurationSteps int   // note length in 16th-note steps (minimum 1)
	Velocity      uint8 // 0-127
}

// ExampleProfile holds statistical features extracted from example MIDI files.
type ExampleProfile struct {
	StepDensity  [16]float64 // probability that each step is active (0.0-1.0), computed across bars
	AccentRatio  float64     // fraction of active steps with velocity > 90
	GhostRatio   float64     // fraction of active steps with velocity < 60
	SlideRatio   float64     // fraction of notes with DurationSteps > 1
	IntervalHist [12]float64 // note distribution as semitones above the detected root (normalized, sum=1.0)
	AvgDensity   float64     // average number of distinct active steps per 16-step bar
}

// SnippetStep is one step in a concrete pattern template.
type SnippetStep struct {
	Active   bool
	Interval int  // semitones above detected root (0=root, 7=fifth, etc.)
	Accent   bool // velocity > 90
	Ghost    bool // velocity < 60
	Slide    bool // duration > 1 step
}

// PatternSnippet is a concrete 16-step template extracted from one bar of an example file.
type PatternSnippet struct {
	Type  string // "bassline" / "arpeggio" / "melody"
	Steps [16]SnippetStep
}

// ExampleBank is the top-level result produced by the loader.
// It is nil when --examples is not provided — all existing code paths are unchanged.
type ExampleBank struct {
	Profiles     map[string]*ExampleProfile  // "bassline" -> profile
	Snippets     map[string][]PatternSnippet // "bassline" -> up to 3 snippets
	DetectedBPM  float64                     // 0 if not detected from meta-events
	DetectedKey  string                      // "" if not detected from meta-events
	PresentTypes []string                    // types found in the folder
	MissingTypes []string                    // types not found (triggers completion mode)
}
