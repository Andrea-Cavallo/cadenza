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
}

type GenerateResult struct {
	Files   []string `json:"files"`
	Elapsed string   `json:"elapsed"`
	Seed    string   `json:"seed"`
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
