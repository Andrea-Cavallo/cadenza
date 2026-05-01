package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Andrea-Cavallo/cadenza/internal/generator"
	midipkg "github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

func runSinglePartAction(cfg cliConfig, patternType string) {
	if lastRun.ProgCLI != "" && cfg.Progression == "" {
		cfg.Progression = lastRun.ProgCLI
	}
	seedStr := fmt.Sprintf("%d", time.Now().UnixNano())
	if lastRun.Seed != "" {
		seedStr = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	fmt.Printf("\n  %s-> Regenerating %s only%s\n\n", ansiGreen, patternType, ansiReset)
	if err := runSinglePartGeneration(cfg, patternType, seedStr, 1); err != nil {
		fmt.Printf("  %s✗  %s regeneration failed: %v%s\n\n", ansiRed+ansiBold, patternType, err, ansiReset)
	}
}

func runSinglePartGeneration(cfg cliConfig, patternType, seedStr string, varNum int) error {
	if !isCLIPatternType(patternType) {
		return fmt.Errorf("unknown pattern type %q", patternType)
	}
	parsedKey, err := theory.ParseKey(cfg.Key)
	if err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	provider, err := buildProvider(cfg.NoLLM, cfg.ProviderName, cfg.Model, cfg.OllamaURL)
	if err != nil {
		if cfg.Interactive {
			if !handleProviderFailure(&cfg, err) {
				return fmt.Errorf("provider init: %w", err)
			}
			provider, err = buildProvider(cfg.NoLLM, cfg.ProviderName, cfg.Model, cfg.OllamaURL)
		}
		if err != nil {
			return fmt.Errorf("provider init: %w", err)
		}
	}

	var prog theory.ChordProgression
	if cfg.Progression != "" {
		prog, err = parseCustomProgression(cfg.Progression, cfg.Key, cfg.Bars)
		if err != nil {
			return fmt.Errorf("custom progression: %w", err)
		}
	} else {
		prog = theory.SelectProgression(parsedKey.Root, parsedKey.Scale, seedStr)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	reg := styleprofile.NewRegistry()
	rend := renderer.New()
	w := midipkg.NewWriter(cfg.BPM)
	mg := generator.NewMultiGenerator(provider, schema.NewValidator(), reg, rend, w, cfg.OutputDir)
	mg.NoLLM = cfg.NoLLM

	result, err := mg.GeneratePartWithContext(ctx, generator.MusicContext{
		BPM:              cfg.BPM,
		Key:              parsedKey,
		Bars:             cfg.Bars,
		VariationSeed:    seedStr,
		ChordProgression: prog,
		Groove:           cfg.Groove,
		OfflineStyle:     cfg.OfflineStyle,
	}, patternType, varNum)
	if err != nil {
		return err
	}

	if cfg.JSONOutput {
		printGenerationJSON(cfg, seedStr, progressionToCLIString(prog), result.Files, false, patternType)
	} else {
		for _, f := range result.Files {
			abs, _ := filepath.Abs(f)
			fmt.Printf("  %s✓%s  %-13s %s%s%s\n", ansiGreen+ansiBold, ansiReset, trackLabel(f), ansiDim, abs, ansiReset)
		}
		fmt.Printf("\n  %sReplaced part:%s %s  %sSeed:%s %s\n\n", ansiBold, ansiReset, patternType, ansiBold, ansiReset, seedStr)
	}

	lastRun = lastRunInfo{Seed: seedStr, ProgCLI: progressionToCLIString(prog), BPM: cfg.BPM, Key: cfg.Key, Files: result.Files}
	return nil
}

func isCLIPatternType(patternType string) bool {
	return patternType == "bassline" || patternType == "arpeggio" || patternType == "melody"
}

func seedPtrFromString(seed string) (*uint64, bool) {
	parsed, err := strconv.ParseUint(seed, 10, 64)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func adjacentKey(keyStr string) string {
	k, err := theory.ParseKey(keyStr)
	if err != nil {
		return keyStr
	}
	root := transposeRoot(k.Root, 7)
	switch k.Mode {
	case "minor":
		return root + "m"
	case "major":
		return root
	default:
		return root + "-" + k.Mode
	}
}

func transposeProgressionCLI(progression, oldKey, newKey string) string {
	oldParsed, oldErr := theory.ParseKey(oldKey)
	newParsed, newErr := theory.ParseKey(newKey)
	if oldErr != nil || newErr != nil || progression == "" {
		return progression
	}
	oldMidi, err := theory.NoteToMIDI(oldParsed.Root + "4")
	if err != nil {
		return progression
	}
	newMidi, err := theory.NoteToMIDI(newParsed.Root + "4")
	if err != nil {
		return progression
	}
	interval := newMidi - oldMidi
	parts := strings.Split(progression, "-")
	for i, part := range parts {
		root, quality, err := theory.ParseChordName(strings.TrimSpace(part))
		if err != nil {
			continue
		}
		parts[i] = transposeRoot(root, interval) + theory.QualitySuffix(quality)
	}
	return strings.Join(parts, "-")
}

func transposeRoot(root string, semitones int) string {
	midi, err := theory.NoteToMIDI(root + "4")
	if err != nil {
		return root
	}
	note := theory.MIDIToNote(midi + semitones)
	return theory.NoteNameOnly(note)
}
