package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// ANSI escape codes.
const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiDim     = "\033[2m"
	ansiRed     = "\033[31m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiMagenta = "\033[35m"
	ansiCyan    = "\033[36m"
	ansiWhite   = "\033[97m"
)

const sepLine = "  " + ansiCyan +
	"──────────────────────────────────────────────────────" +
	ansiReset

// cliConfig holds all generation parameters collected from the user.
type cliConfig struct {
	BPM          float64
	Key          string
	OutputDir    string
	NoLLM        bool
	ProviderName string
	Model        string
}

var stdinReader = bufio.NewReader(os.Stdin)

func ask(label string) string {
	fmt.Printf("  %s%s%s › ", ansiBold, label, ansiReset)
	line, _ := stdinReader.ReadString('\n')
	return strings.TrimSpace(line)
}

func askDefault(label, def string) string {
	fmt.Printf("  %s%s%s [%s%s%s] › ", ansiBold, label, ansiReset, ansiCyan, def, ansiReset)
	line, _ := stdinReader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// bpmGenre returns a genre label for the given BPM value.
func bpmGenre(bpm float64) string {
	switch {
	case bpm < 100:
		return "Downtempo / Ambient"
	case bpm < 115:
		return "Deep House / Nu-Disco"
	case bpm < 122:
		return "Tech House / Organic House"
	case bpm < 130:
		return "Progressive House / Melodic Techno"
	case bpm < 138:
		return "Peak Time Techno"
	default:
		return "Hard Techno / Industrial"
	}
}

// modeLabel returns a display-friendly scale name for producers.
func modeLabel(mode string) string {
	switch mode {
	case "major":
		return "Major  (Ionian)"
	case "minor":
		return "Natural Minor  (Aeolian)"
	default:
		return mode
	}
}

func scaleNoteString(root, scaleType string) string {
	notes, err := theory.ScaleNotes(root, scaleType)
	if err != nil {
		return ""
	}
	return strings.Join(notes, "  ")
}

func printBanner() {
	fmt.Println()
	fmt.Println(sepLine)
	fmt.Println()
	fmt.Printf("    %s▁▂▄█▄▂▁%s  %sL L M I D I - G E N%s  %s▁▂▄█▄▂▁%s\n",
		ansiCyan, ansiReset, ansiBold+ansiWhite, ansiReset, ansiCyan, ansiReset)
	fmt.Println()
	fmt.Printf("    %sAI Composition Engine for Electronic Music%s\n", ansiDim, ansiReset)
	fmt.Printf("    %sProgressive House  ·  Melodic Techno  ·  Peak Techno%s\n", ansiDim, ansiReset)
	fmt.Println()
	fmt.Println(sepLine)
	fmt.Println()
}

// selectMode asks the user for session mode (AI / offline / quit).
// Returns false when the user quits.
func selectMode(cfg *cliConfig) bool {
	fmt.Printf("  %sSESSION MODE%s\n\n", ansiDim+ansiWhite, ansiReset)
	fmt.Printf("  %s[1]%s  %sAI Generation%s      — LLM composes motifs, phrase arcs & evolution\n",
		ansiYellow+ansiBold, ansiReset, ansiBold, ansiReset)
	fmt.Printf("  %s[2]%s  %sOffline / Mock%s      — deterministic patterns, no API key needed\n",
		ansiYellow+ansiBold, ansiReset, ansiBold, ansiReset)
	fmt.Printf("  %s[q]%s  Exit\n\n", ansiYellow+ansiBold, ansiReset)

	for {
		choice := ask("Mode")
		switch choice {
		case "1":
			cfg.NoLLM = false
			return true
		case "2":
			cfg.NoLLM = true
			return true
		case "q", "Q":
			fmt.Printf("\n  %s✕  Session closed.%s\n\n", ansiDim, ansiReset)
			return false
		default:
			fmt.Printf("  %s→ Enter 1, 2, or q%s\n", ansiRed, ansiReset)
		}
	}
}

// selectLLMEngine asks which AI engine to use and verifies the API key.
// Returns false when the user wants to abort entirely.
func selectLLMEngine(cfg *cliConfig) bool {
	fmt.Printf("  %sAI ENGINE%s\n\n", ansiDim+ansiWhite, ansiReset)
	fmt.Printf("  %s[1]%s  %sClaude%s   (Anthropic)  — best musical coherence & variation depth\n",
		ansiYellow+ansiBold, ansiReset, ansiBold, ansiReset)
	fmt.Printf("  %s[2]%s  %sOllama%s   (local)       — runs fully offline, no data sent externally\n\n",
		ansiYellow+ansiBold, ansiReset, ansiBold, ansiReset)

	for {
		choice := ask("Engine")
		switch choice {
		case "1", "":
			cfg.ProviderName = "claude"
			cfg.Model = "claude-opus-4-7"
		case "2":
			cfg.ProviderName = "ollama"
			cfg.Model = "qwen2.5:7b"
		default:
			fmt.Printf("  %s→ Enter 1 or 2%s\n", ansiRed, ansiReset)
			continue
		}
		break
	}
	fmt.Println()

	if cfg.ProviderName == "claude" {
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			fmt.Printf("  %s✕  ANTHROPIC_API_KEY is not set%s\n", ansiRed+ansiBold, ansiReset)
			fmt.Printf("  %s   export ANTHROPIC_API_KEY=sk-ant-...%s\n\n", ansiDim, ansiReset)
			for {
				ans := strings.ToLower(ask("Continue in offline mode instead? [y/N]"))
				switch ans {
				case "y", "yes":
					cfg.NoLLM = true
					fmt.Printf("\n  %s→ Switched to offline mode%s\n\n", ansiGreen, ansiReset)
					return true
				case "", "n", "no":
					fmt.Printf("\n  %sSet the env var and retry.%s\n\n", ansiDim, ansiReset)
					return false
				default:
					fmt.Printf("  %s→ y or n%s\n", ansiRed, ansiReset)
				}
			}
		}
		masked := apiKey[:min(8, len(apiKey))] + "..."
		fmt.Printf("  %s✓  ANTHROPIC_API_KEY%s  %s%s%s\n", ansiGreen+ansiBold, ansiReset, ansiDim, masked, ansiReset)
		cfg.Model = askDefault("Model", cfg.Model)
		fmt.Println()
	} else {
		cfg.Model = askDefault("Model", cfg.Model)
		fmt.Println()
	}
	return true
}

// selectTempo asks for BPM and shows genre context.
func selectTempo(cfg *cliConfig) {
	fmt.Printf("  %sTEMPO%s\n\n", ansiDim+ansiWhite, ansiReset)
	ranges := [][2]string{
		{" 80 – 100", "Downtempo / Ambient"},
		{"100 – 115", "Deep House / Nu-Disco"},
		{"115 – 122", "Tech House / Organic House"},
		{"122 – 130", "Progressive House / Melodic Techno"},
		{"130 – 138", "Peak Time Techno"},
		{"138 – 150", "Hard Techno / Industrial"},
	}
	for _, r := range ranges {
		fmt.Printf("  %s%s%s  %s\n", ansiDim, r[0], ansiReset, r[1])
	}
	fmt.Println()

	for {
		raw := askDefault("BPM (80–150)", "122")
		bpm, err := strconv.ParseFloat(raw, 64)
		if err != nil || bpm < 80 || bpm > 150 {
			fmt.Printf("  %s→ Enter a number between 80 and 150%s\n", ansiRed, ansiReset)
			continue
		}
		cfg.BPM = bpm
		fmt.Printf("  %s→ %.0f BPM  ·  %s%s\n\n", ansiGreen+ansiBold, bpm, bpmGenre(bpm), ansiReset)
		break
	}
}

// selectKey asks for the musical key and shows scale notes on confirmation.
func selectKey(cfg *cliConfig) {
	fmt.Printf("  %sKEY  &  SCALE%s\n\n", ansiDim+ansiWhite, ansiReset)
	fmt.Printf("  %sMajor:%s  C  D  E  F  G  A  B   F#  Bb  Eb  Ab  Db  Gb\n", ansiDim, ansiReset)
	fmt.Printf("  %sMinor:%s  Am  Dm  Em  Bm  F#m  C#m  Gm  Cm  Fm  Bbm  Abm\n\n", ansiDim, ansiReset)

	for {
		raw := askDefault("Key", "Am")
		k, err := theory.ParseKey(raw)
		if err != nil {
			fmt.Printf("  %s→ %v%s\n", ansiRed, err, ansiReset)
			continue
		}
		cfg.Key = raw
		scale := scaleNoteString(k.Root, k.Scale)
		fmt.Printf("  %s→ %s%s  %s%s%s\n",
			ansiGreen+ansiBold, raw, ansiReset,
			ansiDim, modeLabel(k.Mode), ansiReset)
		fmt.Printf("  %s  ♩  %s%s%s\n\n", ansiGreen, ansiCyan+ansiBold, scale, ansiReset)
		break
	}
}

// printSummary renders the pre-render session overview.
func printSummary(cfg cliConfig) {
	k, _ := theory.ParseKey(cfg.Key)
	scale := scaleNoteString(k.Root, k.Scale)

	engineStr := "Offline  (deterministic mock)"
	if !cfg.NoLLM {
		engineStr = cfg.ProviderName + "  /  " + cfg.Model
	}

	fmt.Println(sepLine)
	fmt.Println()
	fmt.Printf("    %s%-9s%s  %s%.0f BPM%s    %s%s%s\n",
		ansiBold, "TEMPO", ansiReset,
		ansiBold+ansiWhite, cfg.BPM, ansiReset,
		ansiDim, bpmGenre(cfg.BPM), ansiReset)
	fmt.Printf("    %s%-9s%s  %s%-7s%s    %s%s%s\n",
		ansiBold, "KEY", ansiReset,
		ansiBold+ansiWhite, cfg.Key, ansiReset,
		ansiDim, modeLabel(k.Mode), ansiReset)
	fmt.Printf("    %s%-9s%s  %s%s%s\n",
		ansiBold, "SCALE", ansiReset,
		ansiCyan, scale, ansiReset)
	fmt.Println()
	fmt.Printf("    %s%-9s%s  %s\n", ansiBold, "ENGINE", ansiReset, engineStr)
	fmt.Printf("    %s%-9s%s  %s/\n", ansiBold, "OUTPUT", ansiReset, cfg.OutputDir)
	fmt.Println()
	fmt.Println(sepLine)
	fmt.Println()
}

// runInteractiveCLI presents the full producer-facing TUI and returns the
// collected config. Returns ok=false when the user cancels or aborts.
func runInteractiveCLI() (cliConfig, bool) {
	printBanner()

	cfg := cliConfig{}

	if !selectMode(&cfg) {
		return cfg, false
	}
	fmt.Println()

	if !cfg.NoLLM {
		if !selectLLMEngine(&cfg) {
			return cfg, false
		}
	}

	selectTempo(&cfg)
	selectKey(&cfg)

	fmt.Printf("  %sOUTPUT%s\n\n", ansiDim+ansiWhite, ansiReset)
	cfg.OutputDir = askDefault("Export directory", "output")
	fmt.Println()

	printSummary(cfg)

	for {
		ans := strings.ToLower(ask("Render session? [Y/n]"))
		switch ans {
		case "", "y", "yes":
			fmt.Println()
			return cfg, true
		case "n", "no":
			fmt.Printf("\n  %s✕  Session cancelled.%s\n\n", ansiDim, ansiReset)
			return cfg, false
		default:
			fmt.Printf("  %s→ y or n%s\n", ansiRed, ansiReset)
		}
	}
}
