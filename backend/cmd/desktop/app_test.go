package main

import (
	"context"
	"os"
	"testing"
)

func TestGetProvidersContainsOffline(t *testing.T) {
	app := NewApp()
	app.ctx = context.Background()
	providers := app.GetProviders()
	if len(providers) == 0 {
		t.Fatal("expected at least one provider")
	}
	found := false
	for _, p := range providers {
		if p == "offline" {
			found = true
		}
	}
	if !found {
		t.Error("expected offline in providers list")
	}
}

func TestStartupLoadsModelsAndStoresContext(t *testing.T) {
	app := NewApp()
	app.outputDir = t.TempDir()
	defer func() { _ = app.closeLogger() }()
	ctx := context.Background()
	app.startup(ctx)

	if app.ctx != ctx {
		t.Fatal("expected startup context to be stored")
	}
	if app.logPath == "" {
		t.Fatal("expected startup to initialize desktop logger")
	}
	if _, err := os.Stat(app.logPath); err != nil {
		t.Fatalf("expected desktop log file to exist: %v", err)
	}
	if got := app.GetModels("claude"); len(got) == 0 {
		t.Fatal("expected startup to load model catalog")
	}
}

func TestGetConfigDefaultBPMNonZero(t *testing.T) {
	app := NewApp()
	app.ctx = context.Background()
	app.outputDir = t.TempDir()
	cfg := app.GetConfig()
	if cfg.DefaultBPM == 0 {
		t.Error("expected non-zero DefaultBPM")
	}
	if cfg.OutputDir != app.outputDir {
		t.Errorf("expected OutputDir %q, got %q", app.outputDir, cfg.OutputDir)
	}
}

func TestGetModelsClaudeReturnsEntries(t *testing.T) {
	app := NewApp()
	app.ctx = context.Background()
	models := app.GetModels("claude")
	if len(models) == 0 {
		t.Error("expected at least one model for claude")
	}
}

func TestGetProviderStatusOfflineReady(t *testing.T) {
	app := NewApp()

	status := app.GetProviderStatus("offline")
	if !status.Ready {
		t.Fatalf("expected offline ready: %#v", status)
	}
	if !status.AuthConfigured {
		t.Fatal("expected offline auth configured")
	}
}

func TestGetProviderStatusClaudeMissingKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	app := NewApp()

	status := app.GetProviderStatus("claude")
	if status.Ready {
		t.Fatalf("expected claude not ready without key: %#v", status)
	}
	if status.Message == "" || status.SetupHint == "" {
		t.Fatalf("expected actionable status: %#v", status)
	}
}

func TestGetProviderStatusClaudeConfigured(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	app := NewApp()

	status := app.GetProviderStatus("claude")
	if !status.Ready {
		t.Fatalf("expected claude ready with key: %#v", status)
	}
}

func TestGetProviderStatusUnknown(t *testing.T) {
	app := NewApp()

	status := app.GetProviderStatus("bad")
	if status.Ready {
		t.Fatalf("expected unknown provider not ready: %#v", status)
	}
}

func TestGenerateOfflineWritesThreeFiles(t *testing.T) {
	app := NewApp()
	app.outputDir = t.TempDir()
	defer func() { _ = app.closeLogger() }()

	result, err := app.Generate(GenerateRequest{
		BPM:          122,
		Key:          "Am",
		Provider:     "offline",
		NoLLM:        true,
		Bars:         16,
		Groove:       "straight",
		OfflineStyle: "minimal",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if len(result.Files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(result.Files))
	}
	if result.Elapsed == "" {
		t.Fatal("expected elapsed duration")
	}
	if result.Seed == "" {
		t.Fatal("expected seed")
	}
	if len(result.Preview.Patterns) != 3 {
		t.Fatalf("expected 3 preview patterns, got %d", len(result.Preview.Patterns))
	}
	if len(result.Preview.Chords) == 0 {
		t.Fatal("expected chord progression preview")
	}
	if result.Preview.StepsPerBar != 16 {
		t.Fatalf("expected 16 steps per bar, got %d", result.Preview.StepsPerBar)
	}
	for _, path := range result.Files {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected generated file %q: %v", path, err)
		}
	}
}

func TestBuildGenerationPreviewUsesPatternSpecs(t *testing.T) {
	app := NewApp()
	app.outputDir = t.TempDir()
	defer func() { _ = app.closeLogger() }()

	result, err := app.Generate(GenerateRequest{
		BPM:          122,
		Key:          "Am",
		Provider:     "offline",
		NoLLM:        true,
		Bars:         16,
		Groove:       "straight",
		OfflineStyle: "melodic",
		Seed:         "preview-test",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var activeNotes int
	for _, pattern := range result.Preview.Patterns {
		if pattern.Label == "" {
			t.Fatalf("expected label for pattern %#v", pattern)
		}
		for _, step := range pattern.Steps {
			if step.Active && step.MIDI > 0 && step.Note != "" {
				activeNotes++
			}
		}
	}
	if activeNotes == 0 {
		t.Fatal("expected preview to expose active MIDI notes")
	}
}

func TestGenerateRejectsMissingClaudeKeyBeforeFallback(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	app := NewApp()
	app.outputDir = t.TempDir()
	defer func() { _ = app.closeLogger() }()

	_, err := app.Generate(GenerateRequest{
		BPM:      122,
		Key:      "Am",
		Provider: "claude",
		Bars:     16,
	})
	if err == nil {
		t.Fatal("expected preflight provider error")
	}
}

func TestGenerateReturnsProviderError(t *testing.T) {
	app := NewApp()
	app.outputDir = t.TempDir()
	defer func() { _ = app.closeLogger() }()

	_, err := app.Generate(GenerateRequest{
		BPM:      122,
		Key:      "Am",
		Provider: "unknown",
		Bars:     16,
	})
	if err == nil {
		t.Fatal("expected provider error")
	}
}

func TestGenerateReturnsInvalidKeyError(t *testing.T) {
	app := NewApp()
	app.outputDir = t.TempDir()
	defer func() { _ = app.closeLogger() }()

	_, err := app.Generate(GenerateRequest{
		BPM:      122,
		Key:      "not-a-key",
		Provider: "offline",
		NoLLM:    true,
		Bars:     16,
	})
	if err == nil {
		t.Fatal("expected invalid key error")
	}
}

func TestGenerateReturnsOutputDirError(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "not-a-dir")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	app.outputDir = file.Name()
	defer func() { _ = app.closeLogger() }()
	_, err = app.Generate(GenerateRequest{
		BPM:      122,
		Key:      "Am",
		Provider: "offline",
		NoLLM:    true,
		Bars:     16,
	})
	if err == nil {
		t.Fatal("expected output dir error")
	}
}

func TestNormalizeGenerateRequestDefaults(t *testing.T) {
	req := normalizeGenerateRequest(GenerateRequest{})

	if req.BPM != 122 {
		t.Errorf("BPM = %d, want 122", req.BPM)
	}
	if req.Key != "Am" {
		t.Errorf("Key = %q, want Am", req.Key)
	}
	if req.Provider != "offline" {
		t.Errorf("Provider = %q, want offline", req.Provider)
	}
	if req.Bars != 16 {
		t.Errorf("Bars = %d, want 16", req.Bars)
	}
	if req.Groove != "straight" {
		t.Errorf("Groove = %q, want straight", req.Groove)
	}
	if req.OfflineStyle != "melodic" {
		t.Errorf("OfflineStyle = %q, want melodic", req.OfflineStyle)
	}
}

func TestNormalizeGenerateRequestEmDashStyle(t *testing.T) {
	req := normalizeGenerateRequest(GenerateRequest{
		BPM:          128,
		Key:          "Dm",
		Provider:     "offline",
		Bars:         32,
		Groove:       "humanize",
		OfflineStyle: "—",
	})
	if req.OfflineStyle != "melodic" {
		t.Errorf("OfflineStyle = %q, want melodic", req.OfflineStyle)
	}
	if req.BPM != 128 || req.Bars != 32 || req.Groove != "humanize" {
		t.Fatalf("non-zero request fields should be preserved: %#v", req)
	}
}

func TestBuildProviderOfflineAndUnknown(t *testing.T) {
	prov, err := buildProvider(true, "claude", "")
	if err != nil {
		t.Fatalf("offline buildProvider error: %v", err)
	}
	if prov.Name() != "mock" {
		t.Fatalf("expected mock provider, got %q", prov.Name())
	}

	if _, err := buildProvider(false, "unknown", ""); err == nil {
		t.Fatal("expected unknown provider error")
	}
}
