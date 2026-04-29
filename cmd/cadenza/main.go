package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/Andrea-Cavallo/cadenza/internal/generator"
	"github.com/Andrea-Cavallo/cadenza/internal/llm"
	midipkg "github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

var version = "dev"

func main() {
	bpmFlag := flag.Float64("bpm", 0, "Tempo in BPM (80-150)")
	keyFlag := flag.String("key", "", "Musical key (e.g. Am, D, F#m, Bb)")
	outputFlag := flag.String("output", "output", "Output directory")
	noLLMFlag := flag.Bool("no-llm", false, "Offline deterministic generation")
	providerFlag := flag.String("provider", "claude", "LLM provider: claude, ollama")
	modelFlag := flag.String("model", "", "Model name (default: claude-opus-4-7 or qwen2.5:7b)")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("cadenza %s\n", version)
		os.Exit(0)
	}

	setupLogger(*outputFlag)

	flagMode := *bpmFlag != 0 || *keyFlag != ""
	if flagMode {
		if *bpmFlag == 0 || *keyFlag == "" {
			fmt.Fprintln(os.Stderr, "Usage: cadenza --bpm 122 --key Am [--output dir] [--no-llm] [--provider claude|ollama] [--model name]")
			os.Exit(1)
		}
		cfg := cliConfig{
			BPM:          *bpmFlag,
			Key:          *keyFlag,
			OutputDir:    *outputFlag,
			NoLLM:        *noLLMFlag,
			ProviderName: *providerFlag,
			Model:        *modelFlag,
		}
		if cfg.Model == "" {
			switch cfg.ProviderName {
			case "claude":
				cfg.Model = "claude-opus-4-7"
			case "ollama":
				cfg.Model = "qwen2.5:7b"
			}
		}
		runGeneration(cfg)
	} else {
		// Interactive daemon: loop until the user quits.
		printBanner()
		for {
			cfg, ok := runInteractiveCLI()
			if !ok {
				break
			}
			runGeneration(cfg)

			fmt.Println()
			fmt.Printf("  %sGenerate another session?%s [Y/n] › ", ansiBold, ansiReset)
			ans, _ := stdinReader.ReadString('\n')
			ans = strings.TrimSpace(strings.ToLower(ans))
			if ans == "n" || ans == "no" {
				fmt.Printf("\n  %s✕  Session closed.%s\n\n", ansiDim, ansiReset)
				break
			}
			fmt.Println()
		}
	}
}

// setupLogger configures slog to write to both stderr and a log file.
func setupLogger(outputDir string) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		slog.Error("cannot create output dir for log", "err", err)
		return
	}
	logPath := filepath.Join(outputDir, "cadenza.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Error("cannot open log file", "path", logPath, "err", err)
		return
	}
	mw := io.MultiWriter(os.Stderr, f)
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	logger := slog.New(slog.NewTextHandler(mw, opts))
	slog.SetDefault(logger)
	slog.Info("cadenza started", "version", version, "log", logPath, "time", time.Now().Format(time.RFC3339))
}

// runGeneration validates config and renders the 3 MIDI tracks.
func runGeneration(cfg cliConfig) {
	if cfg.BPM < 80 || cfg.BPM > 150 {
		fmt.Fprintf(os.Stderr, "BPM must be between 80 and 150, got %.0f\n", cfg.BPM)
		return
	}
	if _, err := theory.ParseKey(cfg.Key); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid key %q: %v\n", cfg.Key, err)
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	slog.Info("generating", "key", cfg.Key, "bpm", cfg.BPM, "provider", cfg.ProviderName, "no-llm", cfg.NoLLM)

	provider, err := buildProvider(cfg.NoLLM, cfg.ProviderName, cfg.Model)
	if err != nil {
		fmt.Printf("  %s✗  Provider init failed: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return
	}

	v := schema.NewValidator()
	reg := styleprofile.NewRegistry()
	rend := renderer.New()
	w := midipkg.NewWriter(cfg.BPM)

	mg := generator.NewMultiGenerator(provider, v, reg, rend, w, cfg.OutputDir)
	mg.NoLLM = cfg.NoLLM

	fmt.Printf("  %sRendering 3 MIDI tracks...%s\n\n", ansiCyan+ansiBold, ansiReset)
	fmt.Printf("  %s♩  Bassline%s   — root motion & low-end chord anchors\n", ansiDim, ansiReset)
	fmt.Printf("  %s♩  Arpeggio%s   — chord-tone texture & rhythmic groove\n", ansiDim, ansiReset)
	fmt.Printf("  %s♩  Melody%s     — phrase arc, motif evolution & fills\n\n", ansiDim, ansiReset)

	result, err := mg.Generate(ctx, cfg.BPM, cfg.Key)
	if err != nil {
		slog.Error("generation failed", "err", err)
		fmt.Printf("  %s✗  Render failed: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return
	}

	fmt.Println(sepLine)
	fmt.Println()
	for _, f := range result.Files {
		rel, _ := filepath.Rel(cfg.OutputDir, f)
		label := trackLabel(rel)
		fmt.Printf("  %s✓%s  %-13s %s%s%s\n", ansiGreen+ansiBold, ansiReset, label, ansiDim, rel, ansiReset)
	}
	fmt.Println()
	fmt.Printf("  %sSeed:%s  %s\n", ansiBold, ansiReset, result.VariationSeed)
	fmt.Println()
	fmt.Println(sepLine)
	fmt.Println()
	fmt.Printf("  %sImport all 3 files to separate MIDI tracks / channels in your DAW.%s\n", ansiDim, ansiReset)
	fmt.Printf("  %sTune BPM to %.0f — each file shares the same chord progression.%s\n\n", ansiDim, cfg.BPM, ansiReset)

	slog.Info("session complete", "files", len(result.Files), "seed", result.VariationSeed)
}

// trackLabel extracts a short track-type label from a filename.
func trackLabel(filename string) string {
	switch {
	case strings.Contains(filename, "bassline"):
		return "[Bassline]"
	case strings.Contains(filename, "arpeggio"):
		return "[Arpeggio]"
	case strings.Contains(filename, "melody"):
		return "[Melody]"
	default:
		return "[Track]"
	}
}

func buildProvider(noLLM bool, providerName, model string) (llm.Provider, error) {
	if noLLM {
		slog.Info("using offline mode (no LLM)")
		return &llm.MockProvider{}, nil
	}
	switch providerName {
	case "claude":
		return llm.NewClaudeProvider(model)
	case "ollama":
		return llm.NewOllamaProvider("http://localhost:11434", model), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", providerName)
	}
}
