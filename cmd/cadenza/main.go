package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/Andrea-Cavallo/cadenza/internal/config"
	"github.com/Andrea-Cavallo/cadenza/internal/generator"
	midipkg "github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

var version = "dev"

func main() {
	// Load config from cadenza.yaml / env vars via Viper
	appCfg, cfgErr := config.Load()
	if cfgErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: config load: %v (using defaults)\n", cfgErr)
		appCfg = &config.AppConfig{}
		appCfg.Audio.BPM = 122
		appCfg.Audio.Key = "Am"
		appCfg.Audio.Bars = 16
		appCfg.Audio.Variations = 1
		appCfg.Audio.Groove = "straight"
		appCfg.LLM.Provider = "claude"
		appCfg.LLM.Model = "claude-opus-4-7"
		appCfg.Output.Dir = "output"
		appCfg.Logging.Level = "info"
		appCfg.Logging.Format = "text"
		appCfg.Logging.File = "cadenza.log"
	}

	// CLI flags override config values
	bpmFlag := flag.Float64("bpm", 0, "Tempo in BPM (80-150)")
	keyFlag := flag.String("key", "", "Musical key (e.g. Am, D, F#m, Bb)")
	outputFlag := flag.String("output", appCfg.Output.Dir, "Output directory")
	noLLMFlag := flag.Bool("no-llm", false, "Offline deterministic generation")
	providerFlag := flag.String("provider", appCfg.LLM.Provider, "LLM provider: claude, ollama, openai, gemini")
	modelFlag := flag.String("model", "", "Model name (overrides config)")
	versionFlag := flag.Bool("version", false, "Print version and exit")

	seedFlag := flag.Uint64("seed", 0, "Deterministic seed (0 = random, printed to stdout)")
	singleFileFlag := flag.Bool("single-file", false, "Output MIDI Type-1 with 3 tracks in one file")
	barsFlag := flag.Int("bars", appCfg.Audio.Bars, "Number of bars (powers of 2: 16, 32, 64, 128)")
	progressionFlag := flag.String("progression", "", "Custom chord progression (e.g. \"Am-F-C-G\")")
	drumsFlag := flag.Bool("drums", false, "Add drum pattern generator (kick/clap/hihat on CH10)")
	variationsFlag := flag.Int("variations", appCfg.Audio.Variations, "Generate N versions with incremental seeds")
	grooveFlag := flag.String("groove", appCfg.Audio.Groove, "Timing preset: straight, mpc60, linndrum, humanize")
	dumpSpecFlag := flag.String("dump-spec", "", "Dump PatternSpec YAML to directory")
	fromSpecFlag := flag.String("from-spec", "", "Re-render from PatternSpec file (bypasses LLM)")

	watchFlag := flag.Bool("watch", false, "Watch mode: stay in loop, Enter generates new variation, 'q' exits")
	dryRunFlag := flag.Bool("dry-run", false, "Execute pipeline without writing files, print summary")
	devFlag := flag.Bool("dev", false, "Interactive dev mode REPL")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("cadenza %s\n", version)
		os.Exit(0)
	}

	if *devFlag {
		setupLogger(*outputFlag, appCfg)
		runDevMode()
		return
	}

	setupLogger(*outputFlag, appCfg)

	slog.Debug("configuration loaded",
		"env", appCfg.App.Env,
		"audio.bars", appCfg.Audio.Bars,
		"audio.bpm", appCfg.Audio.BPM,
		"audio.groove", appCfg.Audio.Groove,
		"llm.provider", appCfg.LLM.Provider,
		"llm.model", appCfg.LLM.Model,
		"output.dir", appCfg.Output.Dir,
		"cache.enabled", appCfg.Cache.Enabled,
	)

	if err := validateFlags(*barsFlag, *variationsFlag, *grooveFlag, *fromSpecFlag, *bpmFlag, *keyFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	flagMode := *bpmFlag != 0 || *keyFlag != "" || *fromSpecFlag != "" || *watchFlag || *noLLMFlag
	if flagMode {
		if *fromSpecFlag == "" && (*bpmFlag == 0 || *keyFlag == "") {
			fmt.Fprintln(os.Stderr, "Usage: cadenza --bpm 122 --key Am [OPTIONS]")
			fmt.Fprintln(os.Stderr, "   or: cadenza --from-spec path/to/spec.yaml [OPTIONS]")
			fmt.Fprintln(os.Stderr, "\nOptions:")
			flag.PrintDefaults()
			os.Exit(1)
		}

		var seedPtr *uint64
		if *seedFlag != 0 {
			seedPtr = seedFlag
		}

		cfg := cliConfig{
			BPM:          *bpmFlag,
			Key:          *keyFlag,
			OutputDir:    *outputFlag,
			NoLLM:        *noLLMFlag,
			ProviderName: *providerFlag,
			Model:        *modelFlag,
			OllamaURL:    appCfg.LLM.OllamaURL,
			Seed:         seedPtr,
			SingleFile:   *singleFileFlag,
			Bars:         *barsFlag,
			Progression:  *progressionFlag,
			Drums:        *drumsFlag,
			Variations:   *variationsFlag,
			Groove:       *grooveFlag,
			DumpSpec:     *dumpSpecFlag,
			FromSpec:     *fromSpecFlag,
			DryRun:       *dryRunFlag,
		}
		if cfg.Model == "" {
			cfg.Model = resolveDefaultModel(cfg.ProviderName, appCfg)
		}

		if *watchFlag {
			runWatchMode(cfg)
			return
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

// validateFlags checks new CLI flags for correctness.
func validateFlags(bars, variations int, groove, fromSpec string, bpm float64, key string) error {
	// Validate bars (must be power of 2)
	if bars != 16 && bars != 32 && bars != 64 && bars != 128 {
		return fmt.Errorf("--bars must be one of: 16, 32, 64, 128 (got %d)", bars)
	}

	// Validate variations
	if variations < 1 || variations > 100 {
		return fmt.Errorf("--variations must be between 1 and 100 (got %d)", variations)
	}

	// Validate groove
	validGrooves := map[string]bool{"straight": true, "mpc60": true, "linndrum": true, "humanize": true}
	if !validGrooves[groove] {
		return fmt.Errorf("--groove must be one of: straight, mpc60, linndrum, humanize (got %q)", groove)
	}

	// If --from-spec is set, --bpm and --key are optional (read from spec)
	// Otherwise they're required (checked in main)
	if fromSpec != "" {
		if _, err := os.Stat(fromSpec); err != nil {
			return fmt.Errorf("--from-spec file not found: %w", err)
		}
	}

	return nil
}

// parseCustomProgression parses a custom chord progression string like "Am-F-C-G"
func parseCustomProgression(progStr, keyStr string, bars int) (theory.ChordProgression, error) {
	if progStr == "" {
		return theory.ChordProgression{}, fmt.Errorf("empty progression string")
	}

	parsedKey, err := theory.ParseKey(keyStr)
	if err != nil {
		return theory.ChordProgression{}, fmt.Errorf("invalid key: %w", err)
	}

	chordStrs := strings.Split(progStr, "-")
	if len(chordStrs) < 2 {
		return theory.ChordProgression{}, fmt.Errorf("progression must have at least 2 chords")
	}
	if len(chordStrs) > 8 {
		return theory.ChordProgression{}, fmt.Errorf("progression cannot have more than 8 chords")
	}

	barsPerChord := bars / len(chordStrs)
	if bars%len(chordStrs) != 0 {
		return theory.ChordProgression{}, fmt.Errorf("bars (%d) must be evenly divisible by chord count (%d)", bars, len(chordStrs))
	}

	prog := theory.ChordProgression{
		Key:    parsedKey.Root,
		Mode:   parsedKey.Mode,
		Chords: make([]theory.ProgressionChord, len(chordStrs)),
	}

	for i, chordStr := range chordStrs {
		chordStr = strings.TrimSpace(chordStr)
		root, quality, err := theory.ParseChordName(chordStr)
		if err != nil {
			return theory.ChordProgression{}, fmt.Errorf("chord %d (%q): %w", i+1, chordStr, err)
		}

		notes, err := theory.ChordNotes(root, quality)
		if err != nil {
			return theory.ChordProgression{}, fmt.Errorf("chord %d (%q): %w", i+1, chordStr, err)
		}

		fromBar := i*barsPerChord + 1
		toBar := (i + 1) * barsPerChord

		prog.Chords[i] = theory.ProgressionChord{
			Root:    root,
			Quality: quality,
			Notes:   notes,
			Bars:    [2]int{fromBar, toBar},
		}
	}

	return prog, nil
}

// runGeneration validates config and renders the MIDI tracks (with support for all new flags).
func runGeneration(cfg cliConfig) {
	// Handle --from-spec mode (re-render from existing spec)
	if cfg.FromSpec != "" {
		runFromSpec(cfg)
		return
	}

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

	// Generate multiple variations if requested
	for varNum := 1; varNum <= cfg.Variations; varNum++ {
		if cfg.Variations > 1 {
			fmt.Printf("\n  %s═══ Variation %d/%d ═══%s\n\n", ansiCyan+ansiBold, varNum, cfg.Variations, ansiReset)
		}

		// Calculate seed for this variation
		var actualSeed uint64
		if cfg.Seed != nil {
			actualSeed = *cfg.Seed + uint64(varNum-1)
		} else {
			actualSeed = uint64(time.Now().UnixNano()) + uint64(varNum-1)
		}
		seedStr := fmt.Sprintf("%d", actualSeed)

		if err := runSingleGeneration(ctx, cfg, varNum, seedStr); err != nil {
			fmt.Printf("  %s✗  Variation %d failed: %v%s\n\n", ansiRed+ansiBold, varNum, err, ansiReset)
		}
	}
}

func runSingleGeneration(ctx context.Context, cfg cliConfig, varNum int, seedStr string) error {
	slog.Info("generating",
		"key", cfg.Key,
		"bpm", cfg.BPM,
		"bars", cfg.Bars,
		"provider", cfg.ProviderName,
		"no-llm", cfg.NoLLM,
		"seed", seedStr,
		"variation", varNum)

	provider, err := buildProvider(cfg.NoLLM, cfg.ProviderName, cfg.Model, cfg.OllamaURL)
	if err != nil {
		return fmt.Errorf("provider init: %w", err)
	}

	v := schema.NewValidator()
	reg := styleprofile.NewRegistry()
	rend := renderer.New()
	w := midipkg.NewWriter(cfg.BPM)

	mg := generator.NewMultiGenerator(provider, v, reg, rend, w, cfg.OutputDir)
	mg.NoLLM = cfg.NoLLM

	// Override chord progression if custom one is provided
	parsedKey, _ := theory.ParseKey(cfg.Key)
	var prog theory.ChordProgression
	if cfg.Progression != "" {
		prog, err = parseCustomProgression(cfg.Progression, cfg.Key, cfg.Bars)
		if err != nil {
			return fmt.Errorf("custom progression: %w", err)
		}
		slog.Info("using custom progression", "progression", cfg.Progression)
	} else {
		prog = theory.SelectProgression(parsedKey.Root, parsedKey.Scale, seedStr)
	}

	slog.Info("chord progression",
		"key", parsedKey.Root+" "+parsedKey.Mode,
		"chords", progressionStringForLog(prog),
	)

	musicCtx := generator.MusicContext{
		BPM:              cfg.BPM,
		Key:              parsedKey,
		Bars:             cfg.Bars,
		VariationSeed:    seedStr,
		ChordProgression: prog,
		Groove:           cfg.Groove,
	}

	fmt.Printf("  %sRendering MIDI tracks...%s\n\n", ansiCyan+ansiBold, ansiReset)
	if cfg.Drums {
		fmt.Printf("  %s♩  Bassline%s   — root motion & low-end chord anchors\n", ansiDim, ansiReset)
		fmt.Printf("  %s♩  Arpeggio%s   — chord-tone texture & rhythmic groove\n", ansiDim, ansiReset)
		fmt.Printf("  %s♩  Melody%s     — phrase arc, motif evolution & fills\n", ansiDim, ansiReset)
		fmt.Printf("  %s♩  Drums%s      — kick, clap, hi-hat pattern (CH10)\n\n", ansiDim, ansiReset)
	} else {
		fmt.Printf("  %s♩  Bassline%s   — root motion & low-end chord anchors\n", ansiDim, ansiReset)
		fmt.Printf("  %s♩  Arpeggio%s   — chord-tone texture & rhythmic groove\n", ansiDim, ansiReset)
		fmt.Printf("  %s♩  Melody%s     — phrase arc, motif evolution & fills\n\n", ansiDim, ansiReset)
	}

	result, err := mg.GenerateWithContext(ctx, musicCtx, cfg.Bars, varNum)
	if err != nil {
		slog.Error("generation failed", "err", err)
		return fmt.Errorf("render: %w", err)
	}

	// REFACTOR.md point 19: Dry-run mode — don't write files, print summary
	if cfg.DryRun {
		fmt.Println(sepLine)
		fmt.Println()
		fmt.Printf("  %s✓  DRY RUN — Pipeline executed successfully%s\n\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %sSeed:%s       %s\n", ansiBold, ansiReset, seedStr)
		fmt.Printf("  %sBars:%s       %d\n", ansiBold, ansiReset, cfg.Bars)
		fmt.Printf("  %sProvider:%s   %s\n", ansiBold, ansiReset, cfg.ProviderName)
		if cfg.Groove != "straight" {
			fmt.Printf("  %sGroove:%s     %s\n", ansiBold, ansiReset, cfg.Groove)
		}
		if cfg.Progression != "" {
			fmt.Printf("  %sChords:%s     %s\n", ansiBold, ansiReset, cfg.Progression)
		}
		fmt.Println()
		fmt.Printf("  %sNo files written (dry-run mode)%s\n", ansiDim, ansiReset)
		fmt.Println()
		fmt.Println(sepLine)
		fmt.Println()
		return nil
	}

	// Dump specs if requested
	if cfg.DumpSpec != "" {
		// TODO: implement spec dumping (result needs to expose specs)
		slog.Info("spec dumping requested but not yet implemented", "dir", cfg.DumpSpec)
	}

	// Add drums if requested
	var drumEvents []midipkg.MIDIEvent
	if cfg.Drums {
		drumEvents = generator.GenerateDrumPattern(cfg.BPM, cfg.Bars, seedStr)
		slog.Info("generated drum pattern", "events", len(drumEvents))
	}

	// Write output (single file or multiple files)
	var outputFiles []string
	if cfg.SingleFile {
		// TODO: implement Type-1 writing
		slog.Warn("--single-file not yet fully implemented, falling back to separate files")
		outputFiles = result.Files
	} else {
		outputFiles = result.Files
	}

	// Print results
	fmt.Println(sepLine)
	fmt.Println()
	for _, f := range outputFiles {
		rel, _ := filepath.Rel(cfg.OutputDir, f)
		label := trackLabel(rel)
		fmt.Printf("  %s✓%s  %-13s %s%s%s\n", ansiGreen+ansiBold, ansiReset, label, ansiDim, rel, ansiReset)
	}
	fmt.Println()
	fmt.Printf("  %sSeed:%s       %s\n", ansiBold, ansiReset, seedStr)
	fmt.Printf("  %sBars:%s       %d\n", ansiBold, ansiReset, cfg.Bars)
	if cfg.Groove != "straight" {
		fmt.Printf("  %sGroove:%s     %s\n", ansiBold, ansiReset, cfg.Groove)
	}
	if cfg.Progression != "" {
		fmt.Printf("  %sChords:%s     %s\n", ansiBold, ansiReset, cfg.Progression)
	}
	fmt.Println()
	fmt.Println(sepLine)
	fmt.Println()
	fmt.Printf("  %sImport all %d file(s) to your DAW.%s\n", ansiDim, len(outputFiles), ansiReset)
	fmt.Printf("  %sTune BPM to %.0f — tracks share the same progression.%s\n\n", ansiDim, cfg.BPM, ansiReset)

	slog.Info("session complete", "files", len(outputFiles), "seed", seedStr, "bars", cfg.Bars)
	return nil
}

func runFromSpec(cfg cliConfig) {
	fmt.Printf("  %sRe-rendering from spec: %s%s\n\n", ansiCyan+ansiBold, cfg.FromSpec, ansiReset)

	spec, err := schema.LoadFromYAML(cfg.FromSpec)
	if err != nil {
		fmt.Printf("  %s✗  Failed to load spec: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return
	}

	// Use BPM and key from spec if not overridden
	bpm := spec.Meta.BPM
	if cfg.BPM != 0 {
		bpm = cfg.BPM
	}

	reg := styleprofile.NewRegistry()
	profile, err := reg.LoadForType(spec.PatternType, spec.StyleProfile)
	if err != nil {
		slog.Warn("style profile not found, using default", "requested", spec.StyleProfile, "error", err)
		profile, err = reg.DefaultForType(spec.PatternType)
		if err != nil {
			fmt.Printf("  %s✗  No profile available for %s: %v%s\n\n", ansiRed+ansiBold, spec.PatternType, err, ansiReset)
			return
		}
	}

	rend := renderer.New()
	events, err := rend.Render(spec, profile)
	if err != nil {
		fmt.Printf("  %s✗  Render failed: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return
	}

	w := midipkg.NewWriter(bpm)
	ts := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("rerendered_%s_%s.mid", spec.PatternType, ts)
	outputPath := filepath.Join(cfg.OutputDir, filename)

	if err := w.WriteFile(outputPath, events); err != nil {
		fmt.Printf("  %s✗  Write failed: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return
	}

	fmt.Println(sepLine)
	fmt.Println()
	fmt.Printf("  %s✓%s  Re-rendered   %s%s%s\n", ansiGreen+ansiBold, ansiReset, ansiDim, outputPath, ansiReset)
	fmt.Println()
	fmt.Println(sepLine)
	fmt.Println()
}

func progressionStringForLog(prog theory.ChordProgression) string {
	if len(prog.Chords) == 0 {
		return ""
	}
	parts := make([]string, len(prog.Chords))
	for i, c := range prog.Chords {
		parts[i] = c.Root + theory.QualitySuffix(c.Quality)
	}
	return strings.Join(parts, " → ")
}

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
