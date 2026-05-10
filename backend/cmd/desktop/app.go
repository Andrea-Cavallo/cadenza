package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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
	ctx          context.Context
	outputDir    string
	logPath      string
	logFile      *os.File
	lastMusicCtx *generator.MusicContext // cached for ReuseChords and per-track regeneration
}

func NewApp() *AppService { return &AppService{} }

func (a *AppService) startup(ctx context.Context) {
	a.ctx = ctx
	a.ensureLogger()
	models.Load("")
}

func (a *AppService) Generate(req GenerateRequest) (GenerateResult, error) {
	start := time.Now()
	a.ensureLogger()

	req = normalizeGenerateRequest(req)
	a.emitProgressStep("provider", fmt.Sprintf("checking %s", req.Provider))
	if err := a.ensureProviderReady(req); err != nil {
		return GenerateResult{}, err
	}

	a.emitLog(fmt.Sprintf("Desktop logger active: %s", a.logPath))
	a.emitProgressStep("provider", "ready")
	a.emitProgressStep("provider", "initializing engine")
	prov, err := buildProvider(req.NoLLM, req.Provider, req.Model)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("provider init: %w", err)
	}
	a.emitProgressStep("provider", "engine initialized")

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
	mg.Sequential = strings.EqualFold(req.Provider, "ollama")

	parsedKey, err := theory.ParseKey(req.Key)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("invalid key %q: %w", req.Key, err)
	}
	a.emitProgressStep("plan", fmt.Sprintf("key parsed: %s", req.Key))

	seed := strings.TrimSpace(req.Seed)
	if seed == "" {
		seed = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Per-track regeneration: reuse the full last context with a new variation seed.
	if req.Track != "" {
		if a.lastMusicCtx == nil {
			return GenerateResult{}, fmt.Errorf("no previous session — generate all tracks first")
		}
		regenCtx := *a.lastMusicCtx
		regenCtx.VariationSeed = seed
		ctx := a.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		a.emitProgressStep("plan", "reusing last chord progression")
		a.emitProgressStep("generate", fmt.Sprintf("regenerating %s PatternSpec", req.Track))
		a.emitProgressStep("render", fmt.Sprintf("rendering %s MIDI", req.Track))
		result, err := mg.GeneratePartWithContext(ctx, regenCtx, req.Track, 1)
		if err != nil {
			return GenerateResult{}, fmt.Errorf("regen %s: %w", req.Track, err)
		}
		a.emitProgressStep("write", fmt.Sprintf("done - %d file written", len(result.Files)))
		for _, w := range mg.Warnings() {
			a.emitProgressStep("warn", w)
		}
		return GenerateResult{
			Files:   result.Files,
			Elapsed: time.Since(start).Round(time.Millisecond).String(),
			Seed:    seed,
			Preview: buildGenerationPreview(regenCtx, result.Specs),
		}, nil
	}

	// Chord progression: reuse last if requested, else generate fresh.
	var prog theory.ChordProgression
	if req.ReuseChords && a.lastMusicCtx != nil {
		prog = a.lastMusicCtx.ChordProgression
		a.emitProgressStep("plan", "keeping last chord progression")
	} else {
		a.emitProgressStep("plan", "selecting chord progression")
		prog = theory.SelectProgression(parsedKey.Root, parsedKey.Scale, seed)
	}
	a.emitProgressStep("generate", fmt.Sprintf("building PatternSpecs for %d bars at %d BPM", req.Bars, req.BPM))

	musicCtx := generator.MusicContext{
		BPM:              float64(req.BPM),
		Key:              parsedKey,
		Bars:             req.Bars,
		VariationSeed:    seed,
		ChordProgression: prog,
		Groove:           req.Groove,
		OfflineStyle:     req.OfflineStyle,
	}
	a.lastMusicCtx = &musicCtx

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	slog.Info("desktop generation started",
		"provider", req.Provider,
		"model", req.Model,
		"no_llm", req.NoLLM,
		"bpm", req.BPM,
		"key", req.Key,
		"bars", req.Bars,
		"groove", req.Groove,
		"offline_style", req.OfflineStyle,
		"seed", seed,
		"reuse_chords", req.ReuseChords,
	)
	a.emitProgressStep("render", "rendering MIDI events")
	a.emitProgressStep("write", "writing MIDI files")
	result, err := mg.GenerateWithContext(ctx, musicCtx, req.Bars, 1)
	if err != nil {
		slog.Error("desktop generation failed", "err", err)
		return GenerateResult{}, fmt.Errorf("generation failed: %w", err)
	}

	a.emitProgressStep("write", fmt.Sprintf("done - %d files written", len(result.Files)))
	for _, w := range mg.Warnings() {
		a.emitProgressStep("warn", w)
	}
	slog.Info("desktop generation finished", "files", len(result.Files), "elapsed", time.Since(start).Round(time.Millisecond).String())

	return GenerateResult{
		Files:   result.Files,
		Elapsed: time.Since(start).Round(time.Millisecond).String(),
		Seed:    seed,
		Preview: buildGenerationPreview(musicCtx, result.Specs),
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

func (a *AppService) GetProviderStatus(provider string) ProviderStatus {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = "offline"
	}
	base := ProviderStatus{
		Provider:     provider,
		DefaultModel: models.DefaultModel(provider),
		Models:       a.GetModels(provider),
	}

	switch provider {
	case "offline":
		base.Ready = true
		base.Installed = true
		base.Running = true
		base.AuthConfigured = true
		base.Message = "Offline engine ready. No API key or network required."
		return base
	case "claude":
		return providerEnvStatus(base, "ANTHROPIC_API_KEY", "Set ANTHROPIC_API_KEY in your environment, then restart Cadenza.")
	case "openai":
		return providerEnvStatus(base, "OPENAI_API_KEY", "Set OPENAI_API_KEY in your environment, then restart Cadenza.")
	case "gemini":
		return providerEnvStatus(base, "GEMINI_API_KEY", "Set GEMINI_API_KEY in your environment, then restart Cadenza.")
	case "ollama":
		return a.ollamaStatus(base)
	default:
		base.Message = fmt.Sprintf("Unknown provider %q.", provider)
		base.SetupHint = "Choose offline, claude, ollama, openai, or gemini."
		return base
	}
}

func (a *AppService) StartOllama() (ProviderStatus, error) {
	if _, err := exec.LookPath("ollama"); err != nil {
		return a.GetProviderStatus("ollama"), fmt.Errorf("ollama executable not found on PATH")
	}
	cmd := exec.Command("ollama", "serve")
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = hiddenWindowSysProcAttr()
	}
	if err := cmd.Start(); err != nil {
		return a.GetProviderStatus("ollama"), fmt.Errorf("start ollama: %w", err)
	}
	time.Sleep(800 * time.Millisecond)
	return a.GetProviderStatus("ollama"), nil
}

// allowedOllamaModels is a safelist to prevent command injection via model names.
var allowedOllamaModels = map[string]bool{
	// Qwen
	"qwen2.5:7b":      true,
	"qwen3-coder:30b": true,
	"qwen3.5:9b":      true,
	"qwen3.6:27b":     true,
	"qwen3.6:35b-a3b": true,
	// Gemma
	"gemma2:2b":      true,
	"gemma4:e2b":     true,
	"gemma4:e4b":     true,
	"gemma4:26b-a4b": true,
	"gemma4:31b":     true,
	// Llama / Mistral / Phi
	"llama3.2:3b": true,
	"mistral:7b":  true,
	"phi3.5:3.8b": true,
	// Other
	"deepseek-r1:7b":   true,
	"nemotron-mini:4b": true,
}

func (a *AppService) PullOllamaModel(model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		model = "qwen2.5:7b"
	}
	if !allowedOllamaModels[model] {
		return fmt.Errorf("model %q is not in the allowed list", model)
	}
	if _, err := exec.LookPath("ollama"); err != nil {
		return fmt.Errorf("ollama not on PATH")
	}
	a.emitProgress(fmt.Sprintf("Pulling %s — this may take several minutes...", model))
	out, err := exec.Command("ollama", "pull", model).CombinedOutput() //nolint:gosec
	if err != nil {
		return fmt.Errorf("pull %s: %w\n%s", model, err, strings.TrimSpace(string(out)))
	}
	a.emitProgress(fmt.Sprintf("Model %s ready.", model))
	return nil
}

func (a *AppService) OpenProviderSetup(provider string) error {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "ollama":
		return openURL("https://ollama.com/download")
	case "claude":
		return openURL("https://console.anthropic.com/settings/keys")
	case "openai":
		return openURL("https://platform.openai.com/api-keys")
	case "gemini":
		return openURL("https://aistudio.google.com/app/apikey")
	default:
		return nil
	}
}

func (a *AppService) GetConfig() AppConfig {
	return AppConfig{
		DefaultBPM:      122,
		DefaultKey:      "Am",
		DefaultProvider: "offline",
		OutputDir:       a.defaultOutputDir(),
	}
}

func (a *AppService) ensureProviderReady(req GenerateRequest) error {
	status := a.GetProviderStatus(req.Provider)
	if status.Ready {
		return nil
	}
	if strings.EqualFold(req.Provider, "ollama") {
		return fmt.Errorf("ollama is not ready: %s", status.Message)
	}
	return fmt.Errorf("%s is not ready: %s", req.Provider, status.Message)
}

func (a *AppService) GetOutputDir() string {
	return a.defaultOutputDir()
}

func (a *AppService) ChooseOutputDir() (string, error) {
	selected, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:            "Choose MIDI Output Folder",
		DefaultDirectory: a.defaultOutputDir(),
	})
	if err != nil {
		return a.defaultOutputDir(), fmt.Errorf("directory dialog: %w", err)
	}
	if selected == "" {
		return a.defaultOutputDir(), nil
	}
	a.outputDir = selected
	return selected, nil
}

func (a *AppService) OpenOutputFolder() error {
	return openFolder(a.defaultOutputDir())
}

func (a *AppService) emitProgress(message string) {
	if !shouldEmitWailsEvents(a.ctx) {
		return
	}
	wailsruntime.EventsEmit(a.ctx, "progress", message)
}

func (a *AppService) emitProgressStep(step, message string) {
	a.emitProgress(fmt.Sprintf("[%s] %s", step, message))
}

func (a *AppService) emitLog(message string) {
	if !shouldEmitWailsEvents(a.ctx) {
		return
	}
	wailsruntime.EventsEmit(a.ctx, "log", message)
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

func (a *AppService) ensureLogger() {
	if a.logPath != "" {
		return
	}
	logPath, logFile, err := setupDesktopLogger(a.defaultOutputDir(), a.ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cadenza desktop logger setup failed: %v\n", err)
		return
	}
	a.logPath = logPath
	a.logFile = logFile
}

func (a *AppService) closeLogger() error {
	if a.logFile == nil {
		return nil
	}
	err := a.logFile.Close()
	a.logFile = nil
	a.logPath = ""
	return err
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

func providerEnvStatus(status ProviderStatus, envName, hint string) ProviderStatus {
	status.Installed = true
	status.Running = true
	status.AuthConfigured = strings.TrimSpace(os.Getenv(envName)) != ""
	status.Ready = status.AuthConfigured
	if status.Ready {
		status.Message = envName + " is configured."
		return status
	}
	status.Message = envName + " is missing."
	status.SetupHint = hint
	return status
}

func (a *AppService) ollamaStatus(status ProviderStatus) ProviderStatus {
	if _, err := exec.LookPath("ollama"); err != nil {
		status.Message = "Ollama is not installed or is not on PATH."
		status.SetupHint = "Install Ollama, then restart Cadenza so the PATH is visible."
		return status
	}
	status.Installed = true

	client := &http.Client{Timeout: 1200 * time.Millisecond}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		status.Message = "Ollama is installed but not running on localhost:11434."
		status.SetupHint = "Click Start Ollama or run `ollama serve`."
		return status
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	status.Running = resp.StatusCode >= 200 && resp.StatusCode < 300
	if !status.Running {
		status.Message = fmt.Sprintf("Ollama responded with HTTP %d.", resp.StatusCode)
		status.SetupHint = "Restart Ollama and try Refresh."
		return status
	}

	var body struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err == nil {
		for _, entry := range body.Models {
			if entry.Name == "" {
				continue
			}
			status.LocalModels = append(status.LocalModels, ModelInfo{ID: entry.Name, Name: entry.Name})
		}
	}

	status.Ready = len(status.LocalModels) > 0
	if status.Ready {
		status.AuthConfigured = true
		status.Message = fmt.Sprintf("Ollama is running with %d local model(s).", len(status.LocalModels))
		return status
	}
	status.Message = "Ollama is running, but no local models are installed."
	status.SetupHint = "Pull a model first, for example `ollama pull qwen2.5:7b`."
	return status
}

func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
