package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Andrea-Cavallo/cadenza/internal/generator"
	midipkg "github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/models"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type AppService struct {
	ctx       context.Context
	outputDir string
}

func NewApp() *AppService { return &AppService{} }

func (a *AppService) startup(ctx context.Context) {
	a.ctx = ctx
	models.Load("")
}

func (a *AppService) Generate(req GenerateRequest) (GenerateResult, error) {
	start := time.Now()

	req = normalizeGenerateRequest(req)
	a.emitProgress("Initializing provider...")
	prov, err := buildProvider(req.NoLLM, req.Provider, req.Model)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("provider init: %w", err)
	}

	outputDir := a.defaultOutputDir()
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return GenerateResult{}, fmt.Errorf("output dir: %w", err)
	}

	v := schema.NewValidator()
	reg := styleprofile.NewRegistry()
	rend := renderer.New()
	w := midipkg.NewWriter(float64(req.BPM))
	mg := generator.NewMultiGenerator(prov, v, reg, rend, w, outputDir)
	mg.NoLLM = req.NoLLM || strings.EqualFold(req.Provider, "offline")

	parsedKey, err := theory.ParseKey(req.Key)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("invalid key %q: %w", req.Key, err)
	}

	seed := fmt.Sprintf("%d", time.Now().UnixNano())
	prog := theory.SelectProgression(parsedKey.Root, parsedKey.Scale, seed)

	a.emitProgress("Generating chord progression...")
	a.emitProgress(fmt.Sprintf("Rendering %d bars at %d BPM in %s...", req.Bars, req.BPM, req.Key))

	musicCtx := generator.MusicContext{
		BPM:              float64(req.BPM),
		Key:              parsedKey,
		Bars:             req.Bars,
		VariationSeed:    seed,
		ChordProgression: prog,
		Groove:           req.Groove,
		OfflineStyle:     req.OfflineStyle,
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := mg.GenerateWithContext(ctx, musicCtx, req.Bars, 1)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("generation failed: %w", err)
	}

	a.emitProgress(fmt.Sprintf("Done - %d files written.", len(result.Files)))

	return GenerateResult{
		Files:   result.Files,
		Elapsed: time.Since(start).Round(time.Millisecond).String(),
		Seed:    seed,
	}, nil
}

func (a *AppService) GetProviders() []string {
	return []string{"claude", "ollama", "openai", "gemini", "offline"}
}

func (a *AppService) GetModels(provider string) []ModelInfo {
	entries := models.List(provider)
	out := make([]ModelInfo, len(entries))
	for i, e := range entries {
		out[i] = ModelInfo{ID: e.ID, Name: e.Name}
	}
	return out
}

func (a *AppService) GetConfig() AppConfig {
	return AppConfig{
		DefaultBPM:      122,
		DefaultKey:      "Am",
		DefaultProvider: "offline",
		OutputDir:       a.defaultOutputDir(),
	}
}

func (a *AppService) OpenOutputFolder() error {
	return openFolder(a.defaultOutputDir())
}

func (a *AppService) emitProgress(message string) {
	if a.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(a.ctx, "progress", message)
}

func normalizeGenerateRequest(req GenerateRequest) GenerateRequest {
	if req.BPM == 0 {
		req.BPM = 122
	}
	if req.Key == "" {
		req.Key = "Am"
	}
	if req.Provider == "" {
		req.Provider = "offline"
	}
	if req.Bars == 0 {
		req.Bars = 16
	}
	if req.Groove == "" {
		req.Groove = "straight"
	}
	if req.OfflineStyle == "" || req.OfflineStyle == "-" || req.OfflineStyle == "—" {
		req.OfflineStyle = "melodic"
	}
	return req
}

func defaultOutputDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "output"
	}
	return filepath.Join(home, "cadenza-output")
}

func (a *AppService) defaultOutputDir() string {
	if a.outputDir != "" {
		return a.outputDir
	}
	return defaultOutputDir()
}

func openFolder(dir string) error {
	cleanDir := filepath.Clean(dir)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", cleanDir)
	case "darwin":
		cmd = exec.Command("open", cleanDir)
	default:
		cmd = exec.Command("xdg-open", cleanDir)
	}
	return cmd.Start()
}
