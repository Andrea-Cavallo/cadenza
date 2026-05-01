package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

type generationJSONSummary struct {
	Seed         string   `json:"seed"`
	BPM          float64  `json:"bpm"`
	Key          string   `json:"key"`
	Bars         int      `json:"bars"`
	Provider     string   `json:"provider"`
	NoLLM        bool     `json:"no_llm"`
	Groove       string   `json:"groove"`
	OfflineStyle string   `json:"offline_style,omitempty"`
	Progression  string   `json:"progression,omitempty"`
	Part         string   `json:"part,omitempty"`
	Files        []string `json:"files,omitempty"`
	DryRun       bool     `json:"dry_run,omitempty"`
	Reproduce    string   `json:"reproduce"`
}

func printGenerationJSON(cfg cliConfig, seed, progression string, files []string, dryRun bool, part string) {
	absFiles := make([]string, 0, len(files))
	for _, f := range files {
		abs, err := filepath.Abs(f)
		if err != nil {
			absFiles = append(absFiles, f)
			continue
		}
		absFiles = append(absFiles, abs)
	}
	summary := generationJSONSummary{
		Seed:         seed,
		BPM:          cfg.BPM,
		Key:          cfg.Key,
		Bars:         cfg.Bars,
		Provider:     cfg.ProviderName,
		NoLLM:        cfg.NoLLM,
		Groove:       cfg.Groove,
		OfflineStyle: cfg.OfflineStyle,
		Progression:  progression,
		Part:         part,
		Files:        absFiles,
		DryRun:       dryRun,
		Reproduce:    reproduceCmd(cfg, seed),
	}
	raw, err := json.Marshal(summary)
	if err != nil {
		fmt.Printf(`{"error":"json summary marshal failed","detail":%q}`+"\n", err.Error())
		return
	}
	fmt.Println(string(raw))
}
