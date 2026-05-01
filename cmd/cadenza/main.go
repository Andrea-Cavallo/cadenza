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

// lastRunInfo holds state from the most recent successful generation for post-run actions.
type lastRunInfo struct {
	Seed    string
	ProgCLI string // "Am-F-C-G" format for --progression flag
	BPM     float64
	Key     string
	Files   []string
}

var lastRun lastRunInfo

func main() {
	if handleConfigCommand(os.Args[1:]) {
		return
	}

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
	offlineStyleFlag := flag.String("offline-style", "", "Offline pattern style: hypnotic, driving, minimal, melodic")
	presetFlag := flag.String("preset", "", "Genre preset: progressive-warmup, peak-time-driver, afterhours-hypnotic, festival-melodic")

	watchFlag := flag.Bool("watch", false, "Watch mode: stay in loop, Enter generates new variation, 'q' exits")
	dryRunFlag := flag.Bool("dry-run", false, "Execute pipeline without writing files, print summary")
	jsonFlag := flag.Bool("json", false, "Print machine-readable JSON generation summaries to stdout")
	devFlag := flag.Bool("dev", false, "Interactive dev mode REPL")
	doctorFlag := flag.Bool("doctor", false, "Run diagnostics: Go version, API keys, Ollama, output directory")
	nonInteractiveFlag := flag.Bool("non-interactive", false, "Non-interactive mode: require --bpm and --key, skip TUI")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("cadenza %s\n", version)
		os.Exit(0)
	}

	if *doctorFlag {
		setupLogger(*outputFlag, appCfg)
		runDoctorCheck(*outputFlag)
		return
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

	if err := validateFlags(*barsFlag, *variationsFlag, *grooveFlag, *fromSpecFlag, *offlineStyleFlag, *bpmFlag, *keyFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Apply genre preset before flag-mode detection so preset fills BPM/key
	if *presetFlag != "" {
		presetCfg := cliConfig{Bars: *barsFlag, Variations: *variationsFlag, Groove: *grooveFlag, OutputDir: *outputFlag}
		if !applyGenrePreset(&presetCfg, *presetFlag) {
			fmt.Fprintf(os.Stderr, "Error: unknown preset %q. Valid: progressive-warmup, peak-time-driver, afterhours-hypnotic, festival-melodic\n", *presetFlag)
			os.Exit(1)
		}
		if *bpmFlag != 0 {
			presetCfg.BPM = *bpmFlag
		}
		if *keyFlag != "" {
			presetCfg.Key = *keyFlag
		}
		presetCfg.NoLLM = *noLLMFlag || presetCfg.NoLLM
		presetCfg.ProviderName = *providerFlag
		presetCfg.Model = *modelFlag
		presetCfg.Seed = func() *uint64 {
			if *seedFlag != 0 {
				return seedFlag
			}
			return nil
		}()
		presetCfg.Drums = *drumsFlag
		presetCfg.DryRun = *dryRunFlag
		presetCfg.JSONOutput = *jsonFlag
		if presetCfg.Model == "" {
			presetCfg.Model = resolveDefaultModel(presetCfg.ProviderName, appCfg)
		}
		runGeneration(presetCfg)
		return
	}

	flagMode := *bpmFlag != 0 || *keyFlag != "" || *fromSpecFlag != "" || *watchFlag || *noLLMFlag || *nonInteractiveFlag
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
			JSONOutput:   *jsonFlag,
			OfflineStyle: *offlineStyleFlag,
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
			if !handlePostRunAction(cfg) {
				break
			}
		}
	}
}

func handlePostRunAction(cfg cliConfig) bool {
	for {
		fmt.Println()
		fmt.Printf("  %sNEXT%s\n\n", ansiDim+ansiWhite, ansiReset)
		fmt.Printf("  %s[1]%s  New guided session\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %s[2]%s  Same setup, new seed\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %s[3]%s  Same harmony, new motifs\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %s[4]%s  A/B compare  (two adjacent seeds)\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %s[5]%s  Faster  (+6 BPM)\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %s[6]%s  Slower  (-6 BPM)\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %s[7]%s  Busier  (driving style)\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %s[8]%s  Sparser (hypnotic style)\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %s[9]%s  Same motifs, new key\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %s[b]%s  Regenerate bassline only\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %s[a]%s  Regenerate arpeggio only\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %s[m]%s  Regenerate melody only\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %s[l]%s  Lock progression mode\n", ansiYellow+ansiBold, ansiReset)
		fmt.Printf("  %s[q]%s  Exit\n\n", ansiYellow+ansiBold, ansiReset)

		switch strings.ToLower(ask("Choose")) {
		case "", "1":
			fmt.Println()
			return true

		case "2":
			fmt.Println()
			cfg.Seed = nil
			runGeneration(cfg)

		case "3":
			// Same harmony (lock progression), new motifs (new seed)
			locked := cfg
			locked.Seed = nil
			if lastRun.ProgCLI != "" {
				locked.Progression = lastRun.ProgCLI
			}
			fmt.Printf("\n  %s-> Locked progression: %s%s\n\n", ansiGreen, locked.Progression, ansiReset)
			runGeneration(locked)

		case "4":
			// A/B compare: two seeds, output in sub-folders A and B
			fmt.Printf("\n  %s═══ Variation A ═══%s\n\n", ansiCyan+ansiBold, ansiReset)
			cfgA := cfg
			cfgA.Seed = nil
			cfgA.OutputDir = filepath.Join(cfg.OutputDir, "A")
			runGeneration(cfgA)
			fmt.Printf("\n  %s═══ Variation B ═══%s\n\n", ansiCyan+ansiBold, ansiReset)
			cfgB := cfg
			cfgB.Seed = nil
			cfgB.OutputDir = filepath.Join(cfg.OutputDir, "B")
			runGeneration(cfgB)
			fmt.Printf("  %sA → %s%s\n", ansiDim, cfgA.OutputDir, ansiReset)
			fmt.Printf("  %sB → %s%s\n\n", ansiDim, cfgB.OutputDir, ansiReset)
			fmt.Printf("  %sImport A and B into your DAW and compare side by side.%s\n\n", ansiDim, ansiReset)

		case "5":
			faster := cfg
			faster.Seed = nil
			faster.BPM = clampBPM(cfg.BPM + 6)
			fmt.Printf("\n  %s-> %.0f BPM (+6)%s\n\n", ansiGreen, faster.BPM, ansiReset)
			runGeneration(faster)
			cfg = faster

		case "6":
			slower := cfg
			slower.Seed = nil
			slower.BPM = clampBPM(cfg.BPM - 6)
			fmt.Printf("\n  %s-> %.0f BPM (-6)%s\n\n", ansiGreen, slower.BPM, ansiReset)
			runGeneration(slower)
			cfg = slower

		case "7":
			busier := cfg
			busier.Seed = nil
			busier.OfflineStyle = "driving"
			busier.NoLLM = true
			fmt.Printf("\n  %s-> Driving style (busier, denser patterns)%s\n\n", ansiGreen, ansiReset)
			runGeneration(busier)

		case "8":
			sparser := cfg
			sparser.Seed = nil
			sparser.OfflineStyle = "hypnotic"
			sparser.NoLLM = true
			fmt.Printf("\n  %s-> Hypnotic style (sparse, meditative patterns)%s\n\n", ansiGreen, ansiReset)
			runGeneration(sparser)

		case "9":
			transposed := cfg
			transposed.Key = adjacentKey(cfg.Key)
			if lastRun.Seed != "" {
				if seedPtr, ok := seedPtrFromString(lastRun.Seed); ok {
					transposed.Seed = seedPtr
				}
			}
			if transposed.Progression != "" {
				transposed.Progression = transposeProgressionCLI(transposed.Progression, cfg.Key, transposed.Key)
			}
			fmt.Printf("\n  %s-> Same seed in %s%s\n\n", ansiGreen, transposed.Key, ansiReset)
			runGeneration(transposed)
			cfg = transposed

		case "b", "bass", "bassline":
			runSinglePartAction(cfg, "bassline")

		case "a", "arp", "arpeggio":
			runSinglePartAction(cfg, "arpeggio")

		case "m", "melody":
			runSinglePartAction(cfg, "melody")

		case "l", "lock":
			if lastRun.ProgCLI == "" {
				fmt.Printf("  %s-> No prior progression available to lock%s\n", ansiRed, ansiReset)
				continue
			}
			cfg.Progression = lastRun.ProgCLI
			fmt.Printf("\n  %s-> Progression locked: %s%s\n\n", ansiGreen, cfg.Progression, ansiReset)

		case "q", "quit", "exit":
			fmt.Printf("\n  %sSession closed.%s\n\n", ansiDim, ansiReset)
			return false

		default:
			fmt.Printf("  %s-> Enter 1-9, b, a, m, l, or q%s\n", ansiRed, ansiReset)
		}
	}
}

func clampBPM(bpm float64) float64 {
	if bpm < 80 {
		return 80
	}
	if bpm > 150 {
		return 150
	}
	return bpm
}

// validateFlags checks new CLI flags for correctness.
func validateFlags(bars, variations int, groove, fromSpec, offlineStyle string, bpm float64, key string) error {
	if bars != 16 && bars != 32 && bars != 64 && bars != 128 {
		return fmt.Errorf("--bars must be one of: 16, 32, 64, 128 (got %d)", bars)
	}
	if variations < 1 || variations > 100 {
		return fmt.Errorf("--variations must be between 1 and 100 (got %d)", variations)
	}
	validGrooves := map[string]bool{"straight": true, "mpc60": true, "linndrum": true, "humanize": true}
	if !validGrooves[groove] {
		return fmt.Errorf("--groove must be one of: straight, mpc60, linndrum, humanize (got %q)", groove)
	}
	if offlineStyle != "" {
		validStyles := map[string]bool{"hypnotic": true, "driving": true, "minimal": true, "melodic": true}
		if !validStyles[offlineStyle] {
			return fmt.Errorf("--offline-style must be one of: hypnotic, driving, minimal, melodic (got %q)", offlineStyle)
		}
	}
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
		if cfg.Variations > 1 && !cfg.JSONOutput {
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
			if cfg.JSONOutput {
				fmt.Fprintf(os.Stderr, "variation %d failed: %v\n", varNum, err)
			} else {
				fmt.Printf("  %s✗  Variation %d failed: %v%s\n\n", ansiRed+ansiBold, varNum, err, ansiReset)
			}
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
		if cfg.Interactive {
			if !handleProviderFailure(&cfg, err) {
				return fmt.Errorf("provider init: %w", err)
			}
			provider, err = buildProvider(cfg.NoLLM, cfg.ProviderName, cfg.Model, cfg.OllamaURL)
			if err != nil {
				return fmt.Errorf("provider init after recovery: %w", err)
			}
		} else {
			return fmt.Errorf("provider init: %w", err)
		}
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
		OfflineStyle:     cfg.OfflineStyle,
	}

	if !cfg.JSONOutput {
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

	}

	result, err := mg.GenerateWithContext(ctx, musicCtx, cfg.Bars, varNum)
	if err != nil {
		slog.Error("generation failed", "err", err)
		return fmt.Errorf("render: %w", err)
	}

	// REFACTOR.md point 19: Dry-run mode — don't write files, print summary
	if cfg.DryRun {
		if cfg.JSONOutput {
			printGenerationJSON(cfg, seedStr, progressionToCLIString(prog), nil, true, "")
			return nil
		}
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

	if cfg.JSONOutput {
		printGenerationJSON(cfg, seedStr, progressionToCLIString(prog), outputFiles, false, "")
		lastRun = lastRunInfo{Seed: seedStr, ProgCLI: progressionToCLIString(prog), BPM: cfg.BPM, Key: cfg.Key, Files: outputFiles}
		slog.Info("session complete", "files", len(outputFiles), "seed", seedStr, "bars", cfg.Bars)
		return nil
	}

	// Print results with absolute paths so users can find files easily
	fmt.Println(sepLine)
	fmt.Println()
	for _, f := range outputFiles {
		abs, _ := filepath.Abs(f)
		label := trackLabel(f)
		fmt.Printf("  %s✓%s  %-13s %s%s%s\n", ansiGreen+ansiBold, ansiReset, label, ansiDim, abs, ansiReset)
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
	fmt.Printf("  %sReproduce:%s  %s%s%s\n", ansiBold, ansiReset, ansiCyan, reproduceCmd(cfg, seedStr), ansiReset)
	fmt.Printf("  %s           Run this command again to recreate these exact patterns.%s\n\n", ansiDim, ansiReset)

	lastRun = lastRunInfo{Seed: seedStr, ProgCLI: progressionToCLIString(prog), BPM: cfg.BPM, Key: cfg.Key, Files: outputFiles}
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

// progressionToCLIString converts a chord progression to the "--progression" flag format "Am-F-C-G".
func progressionToCLIString(prog theory.ChordProgression) string {
	parts := make([]string, len(prog.Chords))
	for i, c := range prog.Chords {
		parts[i] = c.Root + theory.QualitySuffix(c.Quality)
	}
	return strings.Join(parts, "-")
}

// reproduceCmd builds the exact CLI command that recreates this generation.
func reproduceCmd(cfg cliConfig, seed string) string {
	var sb strings.Builder
	sb.WriteString("cadenza")
	fmt.Fprintf(&sb, " --bpm %.0f", cfg.BPM)
	fmt.Fprintf(&sb, " --key %s", cfg.Key)
	fmt.Fprintf(&sb, " --seed %s", seed)
	if cfg.NoLLM {
		sb.WriteString(" --no-llm")
	} else {
		fmt.Fprintf(&sb, " --provider %s", cfg.ProviderName)
		if cfg.Model != "" {
			fmt.Fprintf(&sb, " --model %s", cfg.Model)
		}
	}
	if cfg.Bars != 16 {
		fmt.Fprintf(&sb, " --bars %d", cfg.Bars)
	}
	if cfg.Groove != "straight" && cfg.Groove != "" {
		fmt.Fprintf(&sb, " --groove %s", cfg.Groove)
	}
	if cfg.OfflineStyle != "" {
		fmt.Fprintf(&sb, " --offline-style %s", cfg.OfflineStyle)
	}
	if cfg.Progression != "" {
		fmt.Fprintf(&sb, " --progression %s", cfg.Progression)
	}
	return sb.String()
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
