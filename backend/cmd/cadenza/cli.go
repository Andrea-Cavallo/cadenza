package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Andrea-Cavallo/cadenza/internal/models"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// ANSI escape codes — set to empty when NO_COLOR is defined.
var (
	ansiReset   string
	ansiBold    string
	ansiDim     string
	ansiRed     string
	ansiGreen   string
	ansiYellow  string
	ansiMagenta string
	ansiCyan    string
	ansiWhite   string
)

func init() {
	if os.Getenv("NO_COLOR") != "" {
		return
	}
	ansiReset = "\033[0m"
	ansiBold = "\033[1m"
	ansiDim = "\033[2m"
	ansiRed = "\033[31m"
	ansiGreen = "\033[32m"
	ansiYellow = "\033[33m"
	ansiMagenta = "\033[35m"
	ansiCyan = "\033[36m"
	ansiWhite = "\033[97m"
}

var sepLine = "  " + ansiCyan +
	"------------------------------------------------------" +
	ansiReset

// cliConfig holds all generation parameters collected from the user.
type cliConfig struct {
	BPM          float64
	Key          string
	OutputDir    string
	NoLLM        bool
	ProviderName string
	Model        string
	OllamaURL    string
	Seed         *uint64 // nil = random
	SingleFile   bool
	Bars         int
	Progression  string // custom chord progression, e.g. "Am-F-C-G"
	Drums        bool
	Variations   int
	Groove       string
	DumpSpec     string // directory to dump PatternSpec YAML
	FromSpec     string // path to PatternSpec file to re-render
	DryRun       bool   // REFACTOR.md point 19: execute without writing files
	JSONOutput   bool   // machine-readable stdout summary
	OfflineStyle string // hypnotic | driving | minimal | melodic
	Interactive  bool   // true when launched from the interactive TUI
	LLMTemperature float64
	LLMMaxRetries  int
	LLMTimeout     int
}

var stdinReader = bufio.NewReader(os.Stdin)
var onStdinEOF = func() { os.Exit(0) }

func ask(label string) string {
	fmt.Printf("  %s%s%s > ", ansiBold, label, ansiReset)
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		onStdinEOF()
		return ""
	}
	return strings.TrimSpace(line)
}

func askDefault(label, def string) string {
	fmt.Printf("  %s%s%s [%s%s%s] > ", ansiBold, label, ansiReset, ansiCyan, def, ansiReset)
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		onStdinEOF()
		return def
	}
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
		return "Major  (Ionian) — bright, anthemic"
	case "minor":
		return "Natural Minor  (Aeolian) — dark, emotional"
	case "dorian":
		return "Dorian — dark groove with a major IV chord"
	case "phrygian":
		return "Phrygian — tense, Andalusian darkness"
	case "mixolydian":
		return "Mixolydian — optimistic, festival energy"
	case "lydian":
		return "Lydian — ethereal, floating, dreamy"
	default:
		return mode
	}
}

// keyBPMHint returns a typical BPM range for the given mode.
func keyBPMHint(mode string) string {
	switch mode {
	case "minor":
		return "Typical BPM: 120–130 (progressive house, melodic techno)"
	case "major":
		return "Typical BPM: 124–134 (festival anthems, euphoric trance)"
	case "dorian":
		return "Typical BPM: 124–132 (underground techno, deep grooves)"
	case "phrygian":
		return "Typical BPM: 130–140 (dark peak-time techno, industrial)"
	case "mixolydian":
		return "Typical BPM: 122–130 (festival builds, euphoric breakdowns)"
	case "lydian":
		return "Typical BPM: 118–128 (ethereal progressive, ambient techno)"
	default:
		return "Typical BPM: 120–130"
	}
}

// genrePreset maps a preset name to a full set of generation parameters.
type genrePreset struct {
	name         string
	description  string
	key          string
	bpm          float64
	groove       string
	offlineStyle string
}

var genrePresets = []genrePreset{
	{
		name:         "progressive-warmup",
		description:  "Warm melodic build — emotional Dorian groove for set openers",
		key:          "Am-dorian",
		bpm:          122,
		groove:       "mpc60",
		offlineStyle: "melodic",
	},
	{
		name:         "peak-time-driver",
		description:  "Dense peak-time energy — full-power Aeolian floor filler",
		key:          "Am",
		bpm:          130,
		groove:       "straight",
		offlineStyle: "driving",
	},
	{
		name:         "afterhours-hypnotic",
		description:  "Deep afterhours — sparse, meditative, slow-moving Bm textures",
		key:          "Bm",
		bpm:          116,
		groove:       "humanize",
		offlineStyle: "hypnotic",
	},
	{
		name:         "festival-melodic",
		description:  "Festival euphoria — Mixolydian brightness with anthemic drive",
		key:          "A-mixolydian",
		bpm:          126,
		groove:       "mpc60",
		offlineStyle: "melodic",
	},
}

// applyGenrePreset fills cfg from a preset by name. Returns false if the name is unknown.
func applyGenrePreset(cfg *cliConfig, name string) bool {
	for _, p := range genrePresets {
		if p.name != name {
			continue
		}
		cfg.Key = p.key
		cfg.BPM = p.bpm
		cfg.Groove = p.groove
		cfg.OfflineStyle = p.offlineStyle
		return true
	}
	return false
}

// selectGenrePreset shows the preset menu and applies the chosen preset to cfg.
// Returns true if a preset was applied, false if the user skipped.
func selectGenrePreset(cfg *cliConfig) bool {
	fmt.Printf("  %sGENRE PRESET%s\n\n", ansiDim+ansiWhite, ansiReset)
	for i, p := range genrePresets {
		fmt.Printf("  %s[%d]%s  %-24s %s%s%s\n",
			ansiYellow+ansiBold, i+1, ansiReset,
			p.name,
			ansiDim, p.description, ansiReset)
	}
	fmt.Printf("  %s[s]%s  Skip — configure manually\n\n", ansiYellow+ansiBold, ansiReset)

	for {
		raw := ask("Preset (1–4, s to skip)")
		if raw == "s" || raw == "S" || raw == "" {
			fmt.Println()
			return false
		}
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > len(genrePresets) {
			fmt.Printf("  %s-> Enter 1–%d or s%s\n", ansiRed, len(genrePresets), ansiReset)
			continue
		}
		p := genrePresets[n-1]
		applyGenrePreset(cfg, p.name)
		cfg.NoLLM = true
		fmt.Printf("  %s-> %s%s    %s%s%s\n\n",
			ansiGreen+ansiBold, p.name, ansiReset,
			ansiDim, p.description, ansiReset)
		return true
	}
}

// keyMoodDescription returns a producer-friendly description of the key's emotional character.
func keyMoodDescription(root, mode string) string {
	switch mode {
	case "minor":
		return root + " natural minor — dark, introspective, club-friendly. Hallmark of progressive house and melodic techno."
	case "major":
		return root + " major — bright, anthemic, festival energy. Lifts any room."
	case "dorian":
		return root + " Dorian — darker groove with a lifted 6th. Modal backbone of modern underground techno."
	case "phrygian":
		return root + " Phrygian — tense, Andalusian darkness. One flat 2nd rewrites the emotional character entirely."
	case "mixolydian":
		return root + " Mixolydian — optimistic flat 7th. The mode of festival peaks and euphoric breakdowns."
	case "lydian":
		return root + " Lydian — ethereal raised 4th. Floating, cinematic, otherworldly."
	default:
		return root + " " + mode
	}
}

// energyPreset maps an energy level (1–5) to suggested groove and offline style.
type energyPreset struct {
	label        string
	bpmHint      string
	groove       string
	offlineStyle string
}

var energyPresets = [5]energyPreset{
	{label: "Atmospheric / Hypnotic", bpmHint: "88–105 BPM", groove: "humanize", offlineStyle: "hypnotic"},
	{label: "Deep & Melodic", bpmHint: "108–118 BPM", groove: "straight", offlineStyle: "melodic"},
	{label: "Progressive / Peak", bpmHint: "120–128 BPM", groove: "mpc60", offlineStyle: "driving"},
	{label: "Peak Time / Dense", bpmHint: "128–135 BPM", groove: "straight", offlineStyle: "driving"},
	{label: "Industrial / Minimal", bpmHint: "138–150 BPM", groove: "linndrum", offlineStyle: "minimal"},
}

// selectEnergy asks for an energy level and applies groove/offline-style suggestions.
// Pressing Enter skips without changing anything.
func selectEnergy(cfg *cliConfig) {
	fmt.Printf("  %sENERGY  &  VIBE%s\n\n", ansiDim+ansiWhite, ansiReset)
	for i, p := range energyPresets {
		fmt.Printf("  %s[%d]%s  %-26s %s%s%s\n",
			ansiYellow+ansiBold, i+1, ansiReset,
			p.label,
			ansiDim, p.bpmHint, ansiReset)
	}
	fmt.Println()

	for {
		raw := askDefault("Energy level (1-5, Enter = skip)", "")
		if raw == "" {
			fmt.Println()
			return
		}
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 5 {
			fmt.Printf("  %s-> Enter a number between 1 and 5, or press Enter to skip%s\n", ansiRed, ansiReset)
			continue
		}
		preset := energyPresets[n-1]
		if cfg.Groove == "straight" || cfg.Groove == "" {
			cfg.Groove = preset.groove
		}
		if cfg.OfflineStyle == "" {
			cfg.OfflineStyle = preset.offlineStyle
		}
		fmt.Printf("  %s-> Energy %d: %s%s    %s%s%s\n\n",
			ansiGreen+ansiBold, n, preset.label, ansiReset,
			ansiDim, preset.bpmHint, ansiReset)
		return
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
	candidates := []string{"cadenzabanner.txt"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "cadenzabanner.txt"))
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil {
			fmt.Printf("\n%s%s%s\n\n", ansiCyan, string(data), ansiReset)
			return
		}
	}
	fmt.Println()
}

func printInteractiveWelcome() {
	fmt.Printf("  %sStart with a fast offline sketch or walk through a custom session.%s\n", ansiDim, ansiReset)
	fmt.Printf("  %sOffline mode is a first-class workflow here: fast, musical, and reliable.%s\n\n", ansiDim, ansiReset)
}

func wantsQuickStart() bool {
	for {
		ans := strings.ToLower(ask("Quick sketch preset? [y/N]"))
		switch ans {
		case "", "n", "no":
			fmt.Println()
			return false
		case "y", "yes":
			fmt.Printf("  %s-> Using offline quick sketch defaults%s\n\n", ansiGreen, ansiReset)
			return true
		default:
			fmt.Printf("  %s-> y or n%s\n", ansiRed, ansiReset)
		}
	}
}

func applyQuickStartDefaults(cfg *cliConfig) {
	cfg.NoLLM = true
	cfg.BPM = 122
	cfg.Key = "Am"
	cfg.OutputDir = defaultOutputDir()
	cfg.LLMTemperature = 0.3
	cfg.LLMMaxRetries = 3
	cfg.LLMTimeout = 30
}

// selectMode asks the user for session mode (AI / offline / quit).
// Returns false when the user quits.
func selectMode(cfg *cliConfig) bool {
	fmt.Printf("  %sSESSION MODE%s\n\n", ansiDim+ansiWhite, ansiReset)
	fmt.Printf("  %s[1]%s  %sAI Generation%s      - highest variation, provider-backed composition\n",
		ansiYellow+ansiBold, ansiReset, ansiBold, ansiReset)
	fmt.Printf("  %s[2]%s  %sOffline / Mock%s      - fastest path, no API key, excellent for sketching\n",
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
			fmt.Printf("\n  %sSession closed.%s\n\n", ansiDim, ansiReset)
			return false
		default:
			fmt.Printf("  %s-> Enter 1, 2, or q%s\n", ansiRed, ansiReset)
		}
	}
}

// selectLLMEngine asks which AI engine to use and verifies the API key.
// Returns false when the user wants to abort entirely.
func selectLLMEngine(cfg *cliConfig) bool {
	fmt.Printf("  %sAI ENGINE%s\n\n", ansiDim+ansiWhite, ansiReset)
	fmt.Printf("  %s[1]%s  %sClaude%s   (Anthropic)  - best musical coherence & variation depth\n",
		ansiYellow+ansiBold, ansiReset, ansiBold, ansiReset)
	fmt.Printf("  %s[2]%s  %sOllama%s   (local)       - runs fully offline, no data sent externally\n",
		ansiYellow+ansiBold, ansiReset, ansiBold, ansiReset)
	fmt.Printf("  %s[3]%s  %sOpenAI%s   (hosted)      - structured output with cloud setup\n",
		ansiYellow+ansiBold, ansiReset, ansiBold, ansiReset)
	fmt.Printf("  %s[4]%s  %sGemini%s   (hosted)      - fast hosted JSON generation\n\n",
		ansiYellow+ansiBold, ansiReset, ansiBold, ansiReset)
	printProviderAvailability()

	for {
		choice := ask("Engine")
		switch choice {
		case "1", "":
			cfg.ProviderName = "claude"
		case "2":
			cfg.ProviderName = "ollama"
		case "3":
			cfg.ProviderName = "openai"
		case "4":
			cfg.ProviderName = "gemini"
		default:
			fmt.Printf("  %s-> Enter 1, 2, 3, or 4%s\n", ansiRed, ansiReset)
			continue
		}
		break
	}
	fmt.Println()

	cfg.Model = selectModel(cfg.ProviderName)

	if needsHostedAPIKey(cfg.ProviderName) {
		apiKey := providerAPIKey(cfg.ProviderName)
		if apiKey == "" {
			fmt.Printf("  %sX  %s is not set%s\n", ansiRed+ansiBold, providerEnvVar(cfg.ProviderName), ansiReset)
			fmt.Printf("  %s   %s%s\n\n", ansiDim, providerSetupHint(cfg.ProviderName), ansiReset)
			for {
				ans := strings.ToLower(ask("Continue in offline mode instead? [y/N]"))
				switch ans {
				case "y", "yes":
					cfg.NoLLM = true
					fmt.Printf("\n  %s-> Switched to offline mode%s\n\n", ansiGreen, ansiReset)
					return true
				case "", "n", "no":
					fmt.Printf("\n  %sSet the env var and retry.%s\n\n", ansiDim, ansiReset)
					return false
				default:
					fmt.Printf("  %s-> y or n%s\n", ansiRed, ansiReset)
				}
			}
		}
		masked := apiKey[:min(8, len(apiKey))] + "..."
		if !cfg.JSONOutput {
			fmt.Printf("  %sOK  %s%s  %s%s%s\n", ansiGreen+ansiBold, providerEnvVar(cfg.ProviderName), ansiReset, ansiDim, masked, ansiReset)
		}
		cfg.Model = askDefault("Model", cfg.Model)
		fmt.Println()
	} else {
		cfg.Model = askDefault("Model", cfg.Model)
		fmt.Println()
	}
	return true
}

// selectModel shows the numbered model list for the given provider and returns the chosen model ID.
// If the catalog has no models for the provider, falls back to resolveDefaultModel via empty string.
func selectModel(provider string) string {
	list := models.List(provider)
	if len(list) == 0 {
		return models.DefaultModel(provider)
	}

	fmt.Printf("  %sMODEL  (%s)%s\n\n", ansiDim+ansiWhite, provider, ansiReset)
	for i, m := range list {
		marker := "  "
		if m.ID == models.DefaultModel(provider) {
			marker = ansiGreen + "* " + ansiReset
		}
		fmt.Printf("  %s%s[%d]%s  %s\n", marker, ansiYellow+ansiBold, i+1, ansiReset, m.Name)
	}
	fmt.Printf("\n  %s(press Enter for default: %s)%s\n\n", ansiDim, models.DefaultModel(provider), ansiReset)

	raw := ask("Model")
	if raw == "" {
		return models.DefaultModel(provider)
	}
	idx := 0
	if _, err := fmt.Sscanf(raw, "%d", &idx); err == nil && idx >= 1 && idx <= len(list) {
		chosen := list[idx-1].ID
		fmt.Printf("\n  %s-> %s%s\n\n", ansiGreen, chosen, ansiReset)
		return chosen
	}
	// Accept any non-empty string as a custom model ID (not limited to catalog)
	fmt.Printf("\n  %s-> %s%s\n\n", ansiGreen, raw, ansiReset)
	return raw
}

func printProviderAvailability() {
	lines := []string{
		providerAvailabilityLine("Claude", "ANTHROPIC_API_KEY"),
		providerAvailabilityLine("OpenAI", "OPENAI_API_KEY"),
		providerAvailabilityLine("Gemini", "GEMINI_API_KEY"),
		"  " + ansiDim + "Ollama" + ansiReset + "  local provider, usually at http://localhost:11434",
	}
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println()
}

func providerAvailabilityLine(label, envVar string) string {
	if os.Getenv(envVar) != "" {
		return fmt.Sprintf("  %s%s%s  %sconfigured%s", ansiDim, label, ansiReset, ansiGreen, ansiReset)
	}
	return fmt.Sprintf("  %s%s%s  %snot configured%s", ansiDim, label, ansiReset, ansiYellow, ansiReset)
}

func needsHostedAPIKey(provider string) bool {
	return provider == "claude" || provider == "openai" || provider == "gemini"
}

func providerEnvVar(provider string) string {
	switch provider {
	case "claude":
		return "ANTHROPIC_API_KEY"
	case "openai":
		return "OPENAI_API_KEY"
	case "gemini":
		return "GEMINI_API_KEY"
	default:
		return ""
	}
}

func providerAPIKey(provider string) string {
	return os.Getenv(providerEnvVar(provider))
}

func providerSetupHint(provider string) string {
	switch provider {
	case "claude":
		return "macOS/Linux: export ANTHROPIC_API_KEY=sk-ant-...  |  PowerShell: $env:ANTHROPIC_API_KEY=\"sk-ant-...\""
	case "openai":
		return "macOS/Linux: export OPENAI_API_KEY=sk-...  |  PowerShell: $env:OPENAI_API_KEY=\"sk-...\""
	case "gemini":
		return "macOS/Linux: export GEMINI_API_KEY=...  |  PowerShell: $env:GEMINI_API_KEY=\"...\""
	default:
		return ""
	}
}

// selectTempo asks for BPM and shows genre context.
func selectTempo(cfg *cliConfig) {
	fmt.Printf("  %sTEMPO%s\n\n", ansiDim+ansiWhite, ansiReset)
	ranges := [][2]string{
		{" 80 - 100", "Downtempo / Ambient"},
		{"100 - 115", "Deep House / Nu-Disco"},
		{"115 - 122", "Tech House / Organic House"},
		{"122 - 130", "Progressive House / Melodic Techno"},
		{"130 - 138", "Peak Time Techno"},
		{"138 - 150", "Hard Techno / Industrial"},
	}
	for _, r := range ranges {
		fmt.Printf("  %s%s%s  %s\n", ansiDim, r[0], ansiReset, r[1])
	}
	fmt.Println()

	for {
		raw := askDefault("BPM (80-150)", "122")
		bpm, err := strconv.ParseFloat(raw, 64)
		if err != nil || bpm < 80 || bpm > 150 {
			fmt.Printf("  %s-> Enter a number between 80 and 150%s\n", ansiRed, ansiReset)
			continue
		}
		cfg.BPM = bpm
		fmt.Printf("  %s-> %.0f BPM  .  %s%s\n\n", ansiGreen+ansiBold, bpm, bpmGenre(bpm), ansiReset)
		break
	}
}

// selectKey asks for the musical key and shows scale notes on confirmation.
func selectKey(cfg *cliConfig) {
	fmt.Printf("  %sKEY  &  SCALE%s\n\n", ansiDim+ansiWhite, ansiReset)
	fmt.Printf("  %sMajor:%s         C  D  E  F  G  A  B   F#  Bb  Eb  Ab  Db  Gb\n", ansiDim, ansiReset)
	fmt.Printf("  %sMinor:%s         Am  Dm  Em  Bm  F#m  C#m  Gm  Cm  Fm  Bbm  Abm\n", ansiDim, ansiReset)
	fmt.Printf("  %sDorian:%s        Am-dorian  Dm-dorian  Em-dorian  Bm-dorian\n", ansiDim, ansiReset)
	fmt.Printf("  %sPhrygian:%s      Am-phrygian  Em-phrygian  Bm-phrygian\n", ansiDim, ansiReset)
	fmt.Printf("  %sMixolydian:%s    G-mixolydian  A-mixolydian  D-mixolydian\n", ansiDim, ansiReset)
	fmt.Printf("  %sLydian:%s        C-lydian  D-lydian  G-lydian\n\n", ansiDim, ansiReset)

	for {
		raw := askDefault("Key", "Am")
		k, err := theory.ParseKey(raw)
		if err != nil {
			fmt.Printf("  %s-> %v%s\n", ansiRed, err, ansiReset)
			continue
		}
		cfg.Key = raw
		scale := scaleNoteString(k.Root, k.Scale)
		fmt.Printf("  %s-> %s%s  %s%s%s\n",
			ansiGreen+ansiBold, raw, ansiReset,
			ansiDim, modeLabel(k.Mode), ansiReset)
		fmt.Printf("  %s  notes  %s%s%s\n", ansiGreen, ansiCyan+ansiBold, scale, ansiReset)
		fmt.Printf("  %s  %s%s\n", ansiDim, keyMoodDescription(k.Root, k.Mode), ansiReset)
		fmt.Printf("  %s  %s%s\n\n", ansiDim, keyBPMHint(k.Mode), ansiReset)
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
	if cfg.Groove != "straight" && cfg.Groove != "" {
		fmt.Printf("    %s%-9s%s  %s\n", ansiBold, "GROOVE", ansiReset, cfg.Groove)
	}
	if cfg.OfflineStyle != "" {
		fmt.Printf("    %s%-9s%s  %s\n", ansiBold, "STYLE", ansiReset, cfg.OfflineStyle)
	}
	fmt.Printf("    %s%-9s%s  %s\n", ansiBold, "OUTPUT", ansiReset, cfg.OutputDir)
	fmt.Println()
	fmt.Println(sepLine)
	fmt.Println()
}

func defaultOutputDir() string {
	return filepath.Join("output", time.Now().Format("20060102_150405"))
}

func ensureWritableDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create directory %q: %w", dir, err)
	}
	probe := filepath.Join(dir, ".cadenza-write-test")
	if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
		return fmt.Errorf("directory %q is not writable: %w", dir, err)
	}
	_ = os.Remove(probe) // best-effort cleanup
	return nil
}

func chooseOutputDir(cfg *cliConfig) {
	fmt.Printf("  %sOUTPUT%s\n\n", ansiDim+ansiWhite, ansiReset)
	for {
		cfg.OutputDir = askDefault("Export directory", defaultOutputDir())
		if err := ensureWritableDir(cfg.OutputDir); err != nil {
			fmt.Printf("  %s-> %v%s\n", ansiRed, err, ansiReset)
			continue
		}
		fmt.Printf("  %s-> Output ready: %s%s\n\n", ansiGreen, cfg.OutputDir, ansiReset)
		return
	}
}

func confirmInteractiveRender(cfg cliConfig) (cliConfig, bool) {
	for {
		ans := strings.ToLower(ask("Render session? [Y/n]"))
		switch ans {
		case "", "y", "yes":
			fmt.Println()
			return cfg, true
		case "n", "no":
			fmt.Printf("\n  %sSession canceled.%s\n\n", ansiDim, ansiReset)
			return cfg, false
		default:
			fmt.Printf("  %s-> y or n%s\n", ansiRed, ansiReset)
		}
	}
}

// handleProviderFailure shows a recovery menu when a provider fails to initialize.
// Returns true if the user chose to continue (cfg may be modified), false to cancel.
func handleProviderFailure(cfg *cliConfig, originalErr error) bool {
	fmt.Printf("\n  %s✗  Provider error: %v%s\n\n", ansiRed+ansiBold, originalErr, ansiReset)
	fmt.Printf("  %sPROVIDER FAILURE — choose an option:%s\n\n", ansiDim+ansiWhite, ansiReset)
	fmt.Printf("  %s[1]%s  Retry with same provider\n", ansiYellow+ansiBold, ansiReset)
	fmt.Printf("  %s[2]%s  Switch to offline mode\n", ansiYellow+ansiBold, ansiReset)
	fmt.Printf("  %s[3]%s  Switch to a different provider\n", ansiYellow+ansiBold, ansiReset)
	fmt.Printf("  %s[4]%s  Cancel\n\n", ansiYellow+ansiBold, ansiReset)

	for {
		switch ask("Choose") {
		case "1":
			fmt.Printf("  %s-> Retrying with %s...%s\n\n", ansiGreen, cfg.ProviderName, ansiReset)
			return true
		case "2":
			cfg.NoLLM = true
			fmt.Printf("  %s-> Switched to offline mode%s\n\n", ansiGreen, ansiReset)
			return true
		case "3":
			fmt.Println()
			return selectLLMEngine(cfg)
		case "4", "q", "":
			fmt.Printf("\n  %sGeneration canceled.%s\n\n", ansiDim, ansiReset)
			return false
		default:
			fmt.Printf("  %s-> Enter 1–4%s\n", ansiRed, ansiReset)
		}
	}
}

// runInteractiveCLI presents the full producer-facing TUI and returns the
// collected config. Returns ok=false when the user cancels or aborts.
// The banner is printed once by main before the first call.
func runInteractiveCLI() (cliConfig, bool) {
	cfg := cliConfig{
		Bars:        16,
		Variations:  1,
		Groove:      "straight",
		Interactive: true,
	}

	printInteractiveWelcome()
	if wantsQuickStart() {
		applyQuickStartDefaults(&cfg)
		if err := ensureWritableDir(cfg.OutputDir); err != nil {
			fmt.Printf("  %s-> %v%s\n\n", ansiRed, err, ansiReset)
			return cfg, false
		}
		printSummary(cfg)
		return confirmInteractiveRender(cfg)
	}

	// Offer genre presets — if chosen, skip the manual configuration steps
	if selectGenrePreset(&cfg) {
		chooseOutputDir(&cfg)
		printSummary(cfg)
		return confirmInteractiveRender(cfg)
	}

	if !selectMode(&cfg) {
		return cfg, false
	}
	fmt.Println()

	if !cfg.NoLLM {
		if !selectLLMEngine(&cfg) {
			return cfg, false
		}
	}

	selectEnergy(&cfg)
	selectTempo(&cfg)
	selectKey(&cfg)
	chooseOutputDir(&cfg)

	printSummary(cfg)
	return confirmInteractiveRender(cfg)
}
