package main

type GenerateRequest struct {
	BPM          int    `json:"bpm"`
	Key          string `json:"key"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	NoLLM        bool   `json:"noLlm"`
	Bars         int    `json:"bars"`
	Groove       string `json:"groove"`
	OfflineStyle string `json:"offlineStyle"`
	Seed         string `json:"seed"`        // empty = generate new random seed each time
	ReuseChords  bool   `json:"reuseChords"` // keep last chord progression, vary motifs only
	Track        string `json:"track"`       // "bassline"|"arpeggio"|"melody"|"" (all tracks)
}

type GenerateResult struct {
	Files   []string          `json:"files"`
	Elapsed string            `json:"elapsed"`
	Seed    string            `json:"seed"`
	Preview GenerationPreview `json:"preview"`
}

type ExportEditedRequest struct {
	BPM     int               `json:"bpm"`
	Preview GenerationPreview `json:"preview"`
}

type ExportEditedResult struct {
	Files []string `json:"files"`
}

type GenerationPreview struct {
	Bars        int            `json:"bars"`
	StepsPerBar int            `json:"stepsPerBar"`
	Chords      []ChordPreview `json:"chords"`
	Patterns    []TrackPreview `json:"patterns"`
}

type ChordPreview struct {
	Name    string `json:"name"`
	FromBar int    `json:"fromBar"`
	ToBar   int    `json:"toBar"`
}

type TrackPreview struct {
	PatternType string        `json:"patternType"`
	Label       string        `json:"label"`
	Steps       []StepPreview `json:"steps"`
}

type StepPreview struct {
	Step          int    `json:"step"`
	Note          string `json:"note"`
	MIDI          int    `json:"midi"`
	Active        bool   `json:"active"`
	Accent        bool   `json:"accent"`
	Ghost         bool   `json:"ghost"`
	Slide         bool   `json:"slide"`
	Legato        bool   `json:"legato"`
	Staccato      bool   `json:"staccato"`
	DurationSteps int    `json:"durationSteps"`
	Velocity      int    `json:"velocity"`
}

type ModelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AppConfig struct {
	DefaultBPM      int    `json:"defaultBpm"`
	DefaultKey      string `json:"defaultKey"`
	DefaultProvider string `json:"defaultProvider"`
	OutputDir       string `json:"outputDir"`
}

type ProviderStatus struct {
	Provider       string      `json:"provider"`
	Ready          bool        `json:"ready"`
	Installed      bool        `json:"installed"`
	Running        bool        `json:"running"`
	AuthConfigured bool        `json:"authConfigured"`
	Message        string      `json:"message"`
	SetupHint      string      `json:"setupHint"`
	DefaultModel   string      `json:"defaultModel"`
	Models         []ModelInfo `json:"models"`
	LocalModels    []ModelInfo `json:"localModels"`
}
