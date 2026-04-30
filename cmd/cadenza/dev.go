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
	fmt.Printf("\n  %s═══ DEV MODE ═══%s\n\n", ansiMagenta+ansiBold, ansiReset)
	fmt.Printf("  %sInteractive REPL for development and debugging%s\n", ansiDim, ansiReset)
	fmt.Printf("  %sType 'help' for available commands%s\n\n", ansiDim, ansiReset)

	reader := bufio.NewReader(os.Stdin)

	// Shared resources
	var lastSpecs [3]*schema.PatternSpec
	var lastProgression theory.ChordProgression
	cacheInstance := cache.New(30, "./cache")

	ctx := context.Background()

	for {
		fmt.Printf("  %scadenza [dev]%s › ", ansiMagenta+ansiBold, ansiReset)
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("  %s✗  Input error: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
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
			fmt.Printf("\n  %s✕  Dev mode exited.%s\n\n", ansiDim, ansiReset)
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
			fmt.Printf("  %s✗  Unknown command: %s%s\n", ansiRed, cmd, ansiReset)
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

func devGenerate(ctx context.Context, args []string, cacheInstance *cache.Cache) ([3]*schema.PatternSpec, theory.ChordProgression) {
	var cfg cliConfig
	cfg.BPM = 122
	cfg.Key = "Am"
	cfg.OutputDir = "output"
	cfg.ProviderName = "claude"
	cfg.Model = "claude-opus-4-7"
	cfg.Bars = 16

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--bpm":
			if i+1 < len(args) {
				if bpm, err := strconv.ParseFloat(args[i+1], 64); err == nil {
					cfg.BPM = bpm
					i++
				}
			}
		case "--key":
			if i+1 < len(args) {
				cfg.Key = args[i+1]
				i++
			}
		case "--no-llm":
			cfg.NoLLM = true
		case "--provider":
			if i+1 < len(args) {
				cfg.ProviderName = args[i+1]
				i++
			}
		case "--seed":
			if i+1 < len(args) {
				if seed, err := strconv.ParseUint(args[i+1], 10, 64); err == nil {
					cfg.Seed = &seed
					i++
				}
			}
		}
	}

	// Validate
	if cfg.BPM < 80 || cfg.BPM > 150 {
		fmt.Printf("  %s✗  BPM must be 80-150, got %.0f%s\n\n", ansiRed+ansiBold, cfg.BPM, ansiReset)
		return [3]*schema.PatternSpec{}, theory.ChordProgression{}
	}
	if _, err := theory.ParseKey(cfg.Key); err != nil {
		fmt.Printf("  %s✗  Invalid key: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return [3]*schema.PatternSpec{}, theory.ChordProgression{}
	}

	fmt.Printf("\n  %s→ Generating with BPM=%.0f, Key=%s, NoLLM=%v%s\n\n", ansiGreen, cfg.BPM, cfg.Key, cfg.NoLLM, ansiReset)

	provider, err := buildProvider(cfg.NoLLM, cfg.ProviderName, cfg.Model, cfg.OllamaURL)
	if err != nil {
		fmt.Printf("  %s✗  Provider init: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
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
		fmt.Printf("  %s✗  Generation failed: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return [3]*schema.PatternSpec{}, prog
	}

	fmt.Println()
	fmt.Printf("  %s✓  Generated 3 patterns successfully%s\n", ansiGreen+ansiBold, ansiReset)
	for _, f := range result.Files {
		fmt.Printf("     %s%s%s\n", ansiDim, f, ansiReset)
	}
	fmt.Println()

	// Return empty specs for now (would need to expose from generator)
	return [3]*schema.PatternSpec{}, prog
}

func devRender(_ []string, _ [3]*schema.PatternSpec) {
	fmt.Printf("  %s✗  render: not implemented yet%s\n", ansiRed, ansiReset)
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
		fmt.Printf("  %s✗  Usage: validate --spec <file>%s\n\n", ansiRed, ansiReset)
		return
	}

	spec, err := schema.LoadFromYAML(specFile)
	if err != nil {
		fmt.Printf("  %s✗  Load failed: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return
	}

	v := schema.NewValidator()
	if err := v.ValidateWithChords(spec, prog); err != nil {
		fmt.Printf("  %s✗  Validation failed:%s\n", ansiRed+ansiBold, ansiReset)
		fmt.Printf("     %s%v%s\n\n", ansiDim, err, ansiReset)
	} else {
		fmt.Printf("  %s✓  Validation passed%s\n\n", ansiGreen+ansiBold, ansiReset)
	}
}

func devInspect(_ []string, _ [3]*schema.PatternSpec) {
	fmt.Printf("  %s✗  inspect: not implemented yet%s\n", ansiRed, ansiReset)
	fmt.Printf("     Use '--dump-spec' flag to export specs for inspection\n\n")
}

func devChordProgression(args []string) theory.ChordProgression {
	key := "Am"
	seed := "dev-seed"
	var custom string

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--key":
			if i+1 < len(args) {
				key = args[i+1]
				i++
			}
		case "--seed":
			if i+1 < len(args) {
				seed = args[i+1]
				i++
			}
		case "--custom":
			if i+1 < len(args) {
				custom = args[i+1]
				i++
			}
		}
	}

	parsedKey, err := theory.ParseKey(key)
	if err != nil {
		fmt.Printf("  %s✗  Invalid key: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		return theory.ChordProgression{}
	}

	var prog theory.ChordProgression
	if custom != "" {
		prog, err = parseCustomProgression(custom, key, 16)
		if err != nil {
			fmt.Printf("  %s✗  Custom progression parse failed: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
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
		fmt.Printf("  %s✗  Cache not initialized%s\n\n", ansiRed, ansiReset)
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
	fmt.Printf("  %s✗  from-spec: not implemented yet%s\n", ansiRed, ansiReset)
	fmt.Printf("     Use '--from-spec <file>' flag in normal mode\n\n")
}

func devDumpSpec(_ []string, _ [3]*schema.PatternSpec) {
	fmt.Printf("  %s✗  dump-spec: not implemented yet%s\n", ansiRed, ansiReset)
	fmt.Printf("     Use '--dump-spec' flag in normal mode\n\n")
}
