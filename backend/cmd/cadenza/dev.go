package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Andrea-Cavallo/cadenza/internal/cache"
	"github.com/Andrea-Cavallo/cadenza/internal/generator"
	midipkg "github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// runDevMode enters interactive development REPL with commands for inspecting,
// generating, validating, and rendering patterns.
// REFACTOR.md point 20
func runDevMode() {
	fmt.Printf("\n  %s=== DEV MODE ===%s\n\n", ansiMagenta+ansiBold, ansiReset)
	fmt.Printf("  %sInteractive REPL for development and debugging%s\n", ansiDim, ansiReset)
	fmt.Printf("  %sType 'help' for available commands%s\n\n", ansiDim, ansiReset)

	reader := bufio.NewReader(os.Stdin)

	// Shared resources
	var lastSpecs [3]*schema.PatternSpec
	var lastProgression theory.ChordProgression
	cacheInstance := cache.New(30, "./cache")

	ctx := context.Background()

	for {
		fmt.Printf("  %scadenza [dev]%s â€º ", ansiMagenta+ansiBold, ansiReset)
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("  %sâœ—  Input error: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		args := strings.Fields(line)
		cmd := args[0]

		switch cmd {
		case "help", "h":
			printDevHelp()

		case "exit", "quit", "q":
			fmt.Printf("\n  %sX  Dev mode exited.%s\n\n", ansiDim, ansiReset)
			return

		case "generate", "gen":
			lastSpecs, lastProgression = devGenerate(ctx, args, cacheInstance)

		case "render":
			devRender(args, lastSpecs)

		case "validate", "val":
			devValidate(args, lastProgression)

		case "inspect", "i":
			devInspect(args, lastSpecs)

		case "chord-progression", "chords", "cp":
			lastProgression = devChordProgression(args)

		case "cache-info", "cache":
			devCacheInfo(cacheInstance)

		case "from-spec", "load":
			devFromSpec(args, &lastSpecs)

		case "dump-spec", "dump":
			devDumpSpec(args, lastSpecs)

		default:
			fmt.Printf("  %sâœ—  Unknown command: %s%s\n", ansiRed, cmd, ansiReset)
			fmt.Printf("  %s   Type 'help' for available commands%s\n\n", ansiDim, ansiReset)
		}
	}
}

func printDevHelp() {
	fmt.Println()
	fmt.Printf("  %sAVAILABLE COMMANDS%s\n\n", ansiBold+ansiWhite, ansiReset)
	fmt.Printf("  %sgenerate%s --bpm 122 --key Am [--no-llm] [--provider claude] [--seed 123]\n", ansiCyan+ansiBold, ansiReset)
	fmt.Printf("    %sGenerate 3 patterns (bassline, arpeggio, melody) with full pipeline%s\n\n", ansiDim, ansiReset)
	fmt.Printf("  %srender%s --from-spec <file> [--output <dir>]\n", ansiCyan+ansiBold, ansiReset)
	fmt.Printf("    %sRe-render a PatternSpec from YAML/JSON to MIDI%s\n\n", ansiDim, ansiReset)
	fmt.Printf("  %svalidate%s --spec <file>\n", ansiCyan+ansiBold, ansiReset)
	fmt.Printf("    %sValidate a PatternSpec file and show structured errors%s\n\n", ansiDim, ansiReset)
	fmt.Printf("  %sinspect%s --type <bassline|arpeggio|melody> [--step <n>]\n", ansiCyan+ansiBold, ansiReset)
	fmt.Printf("    %sPrint motif steps with notes, velocity, timing, gate (from last generation)%s\n\n", ansiDim, ansiReset)
	fmt.Printf("  %schord-progression%s --key Am [--seed 123] [--custom \"Am-F-C-G\"]\n", ansiCyan+ansiBold, ansiReset)
	fmt.Printf("    %sGenerate and print chord progression%s\n\n", ansiDim, ansiReset)
	fmt.Printf("  %scache-info%s\n", ansiCyan+ansiBold, ansiReset)
	fmt.Printf("    %sShow cache statistics (hit/miss rate, keys)%s\n\n", ansiDim, ansiReset)
	fmt.Printf("  %sdump-spec%s --type <bassline|arpeggio|melody> --output <file>\n", ansiCyan+ansiBold, ansiReset)
	fmt.Printf("    %sDump last generated spec to YAML%s\n\n", ansiDim, ansiReset)
	fmt.Printf("  %sload%s --spec <file>\n", ansiCyan+ansiBold, ansiReset)
	fmt.Printf("    %sLoad a PatternSpec into memory for inspection%s\n\n", ansiDim, ansiReset)
	fmt.Printf("  %shelp%s\n", ansiCyan+ansiBold, ansiReset)
	fmt.Printf("    %sShow this help message%s\n\n", ansiDim, ansiReset)
	fmt.Printf("  %sexit%s / %squit%s / %sq%s\n", ansiCyan+ansiBold, ansiReset, ansiCyan+ansiBold, ansiReset, ansiCyan+ansiBold, ansiReset)
	fmt.Printf("    %sExit dev mode%s\n\n", ansiDim, ansiReset)
}

func parseGenerateFlags(args []string) cliConfig {
	cfg := cliConfig{BPM: 122, Key: "Am", OutputDir: "output", ProviderName: "claude", Model: "claude-opus-4-7", Bars: 16}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--bpm":
			value, ok := flagValue(args, &i)
			if ok {
				setFloatFlag(value, &cfg.BPM)
			}
		case "--key":
			if value, ok := flagValue(args, &i); ok {
				cfg.Key = value
			}
		case "--no-llm":
			cfg.NoLLM = true
		case "--provider":
			if value, ok := flagValue(args, &i); ok {
				cfg.ProviderName = value
			}
		case "--seed":
			value, ok := flagValue(args, &i)
			if ok {
				cfg.Seed = parseUintPtr(value)
			}
		}
	}
	return cfg
}

func flagValue(args []string, idx *int) (string, bool) {
	if *idx+1 >= len(args) {
		return "", false
	}
	*idx++
	return args[*idx], true
}

func setFloatFlag(value string, target *float64) {
	if parsed, err := strconv.ParseFloat(value, 64); err == nil {
		*target = parsed
	}
}

func parseUintPtr(value string) *uint64 {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func devGenerate(ctx context.Context, args []string, cacheInstance *cache.Cache) ([3]*schema.PatternSpec, theory.ChordProgression) {
	cfg := parseGenerateFlags(args)

	// Validate
	if cfg.BPM < 80 || cfg.BPM > 150 {
		fmt.Printf("  %sâœ—  BPM must be 80-150, got %.0f%s\n\n", ansiRed+ansiBold, cfg.BPM, ansiReset)
		return [3]*schema.PatternSpec{}, theory.ChordProgression{}
	}
	if _, err := theory.ParseKey(cfg.Key); err != nil {
		fmt.Printf("  %sâœ—  Invalid key: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return [3]*schema.PatternSpec{}, theory.ChordProgression{}
	}

	fmt.Printf("\n  %sâ†’ Generating with BPM=%.0f, Key=%s, NoLLM=%v%s\n\n", ansiGreen, cfg.BPM, cfg.Key, cfg.NoLLM, ansiReset)

	provider, err := buildProvider(cfg.NoLLM, cfg.ProviderName, cfg.Model, cfg.OllamaURL)
	if err != nil {
		fmt.Printf("  %sâœ—  Provider init: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return [3]*schema.PatternSpec{}, theory.ChordProgression{}
	}

	parsedKey, _ := theory.ParseKey(cfg.Key)
	seed := "dev-seed"
	if cfg.Seed != nil {
		seed = fmt.Sprintf("%d", *cfg.Seed)
	}

	prog := theory.SelectProgression(parsedKey.Root, parsedKey.Scale, seed)
	fmt.Printf("  %sChord Progression:%s %s\n\n", ansiBold, ansiReset, progressionStringForLog(prog))

	v := schema.NewValidator()
	reg := styleprofile.NewRegistry()
	rend := renderer.New()
	w := midipkg.NewWriter(cfg.BPM)

	mg := generator.NewMultiGenerator(provider, v, reg, rend, w, cfg.OutputDir)
	mg.NoLLM = cfg.NoLLM

	musicCtx := generator.MusicContext{
		BPM:              cfg.BPM,
		Key:              parsedKey,
		Bars:             cfg.Bars,
		VariationSeed:    seed,
		ChordProgression: prog,
		Groove:           "straight",
	}

	result, err := mg.GenerateWithContext(ctx, musicCtx, cfg.Bars, 1)
	if err != nil {
		fmt.Printf("  %sâœ—  Generation failed: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return [3]*schema.PatternSpec{}, prog
	}

	fmt.Println()
	fmt.Printf("  %sâœ“  Generated 3 patterns successfully%s\n", ansiGreen+ansiBold, ansiReset)
	for _, f := range result.Files {
		fmt.Printf("     %s%s%s\n", ansiDim, f, ansiReset)
	}
	fmt.Println()

	// Return empty specs for now (would need to expose from generator)
	return [3]*schema.PatternSpec{}, prog
}

func devRender(_ []string, _ [3]*schema.PatternSpec) {
	fmt.Printf("  %sâœ—  render: not implemented yet%s\n", ansiRed, ansiReset)
	fmt.Printf("     Use 'go run ./cmd/cadenza/ --bpm 122 --key Am' for full rendering\n\n")
}

func devValidate(args []string, prog theory.ChordProgression) {
	var specFile string
	for i := 1; i < len(args); i++ {
		if args[i] == "--spec" && i+1 < len(args) {
			specFile = args[i+1]
			break
		}
	}

	if specFile == "" {
		fmt.Printf("  %sâœ—  Usage: validate --spec <file>%s\n\n", ansiRed, ansiReset)
		return
	}

	spec, err := schema.LoadFromYAML(specFile)
	if err != nil {
		fmt.Printf("  %sâœ—  Load failed: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return
	}

	v := schema.NewValidator()
	if err := v.ValidateWithChords(spec, prog); err != nil {
		fmt.Printf("  %sâœ—  Validation failed:%s\n", ansiRed+ansiBold, ansiReset)
		fmt.Printf("     %s%v%s\n\n", ansiDim, err, ansiReset)
	} else {
		fmt.Printf("  %sâœ“  Validation passed%s\n\n", ansiGreen+ansiBold, ansiReset)
	}
}

func devInspect(_ []string, _ [3]*schema.PatternSpec) {
	fmt.Printf("  %sâœ—  inspect: not implemented yet%s\n", ansiRed, ansiReset)
	fmt.Printf("     Use '--dump-spec' flag to export specs for inspection\n\n")
}

func parseChordProgressionFlags(args []string) (key, seed, custom string) {
	key, seed = "Am", "dev-seed"
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--key":
			if value, ok := flagValue(args, &i); ok {
				key = value
			}
		case "--seed":
			if value, ok := flagValue(args, &i); ok {
				seed = value
			}
		case "--custom":
			if value, ok := flagValue(args, &i); ok {
				custom = value
			}
		}
	}
	return
}

func devChordProgression(args []string) theory.ChordProgression {
	key, seed, custom := parseChordProgressionFlags(args)

	parsedKey, err := theory.ParseKey(key)
	if err != nil {
		fmt.Printf("  %sâœ—  Invalid key: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return theory.ChordProgression{}
	}

	var prog theory.ChordProgression
	if custom != "" {
		prog, err = parseCustomProgression(custom, key, 16)
		if err != nil {
			fmt.Printf("  %sâœ—  Custom progression parse failed: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
			return theory.ChordProgression{}
		}
	} else {
		prog = theory.SelectProgression(parsedKey.Root, parsedKey.Scale, seed)
	}

	fmt.Println()
	fmt.Printf("  %sChord Progression%s\n\n", ansiBold+ansiWhite, ansiReset)
	for i, c := range prog.Chords {
		notesStr := strings.Join(c.Notes, ", ")
		fmt.Printf("  %s[%d]%s  bars %d-%d:  %s%s%s  (%s)\n",
			ansiYellow+ansiBold, i+1, ansiReset,
			c.Bars[0], c.Bars[1],
			ansiCyan+ansiBold, c.Root, theory.QualitySuffix(c.Quality),
			notesStr+ansiReset)
	}
	fmt.Println()

	return prog
}

func devCacheInfo(cacheInstance *cache.Cache) {
	if cacheInstance == nil {
		fmt.Printf("  %sâœ—  Cache not initialized%s\n\n", ansiRed, ansiReset)
		return
	}

	stats := cacheInstance.Stats()
	hitRate := 0.0
	total := stats["hits"].(int) + stats["misses"].(int)
	if total > 0 {
		hitRate = float64(stats["hits"].(int)) * 100.0 / float64(total)
	}

	fmt.Println()
	fmt.Printf("  %sCACHE STATISTICS%s\n\n", ansiBold+ansiWhite, ansiReset)
	fmt.Printf("  %sHits:%s       %d\n", ansiBold, ansiReset, stats["hits"])
	fmt.Printf("  %sMisses:%s     %d\n", ansiBold, ansiReset, stats["misses"])
	fmt.Printf("  %sHit Rate:%s   %.1f%%\n", ansiBold, ansiReset, hitRate)
	fmt.Printf("  %sKeys:%s       %d\n\n", ansiBold, ansiReset, stats["keys"])
}

func devFromSpec(_ []string, _ *[3]*schema.PatternSpec) {
	fmt.Printf("  %sâœ—  from-spec: not implemented yet%s\n", ansiRed, ansiReset)
	fmt.Printf("     Use '--from-spec <file>' flag in normal mode\n\n")
}

func devDumpSpec(_ []string, _ [3]*schema.PatternSpec) {
	fmt.Printf("  %sâœ—  dump-spec: not implemented yet%s\n", ansiRed, ansiReset)
	fmt.Printf("     Use '--dump-spec' flag in normal mode\n\n")
}
