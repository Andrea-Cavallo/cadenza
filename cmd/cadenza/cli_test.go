package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/config"
)

func withInput(input string, fn func()) {
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader(input))
	defer func() { stdinReader = old }()
	fn()
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestCLIInputHelpers(t *testing.T) {
	withInput("hello\n", func() {
		if got := ask("Label"); got != "hello" {
			t.Fatalf("unexpected ask result %q", got)
		}
	})
	withInput("\n", func() {
		if got := askDefault("Label", "fallback"); got != "fallback" {
			t.Fatalf("unexpected askDefault fallback %q", got)
		}
	})
	withInput("custom\n", func() {
		if got := askDefault("Label", "fallback"); got != "custom" {
			t.Fatalf("unexpected askDefault explicit %q", got)
		}
	})
}

func TestKeyMoodDescription(t *testing.T) {
	cases := map[string]string{
		"minor":      "dark",
		"major":      "bright",
		"dorian":     "Dorian",
		"phrygian":   "Phrygian",
		"mixolydian": "Mixolydian",
		"lydian":     "Lydian",
		"unknown":    "A unknown",
	}
	for mode, want := range cases {
		got := keyMoodDescription("A", mode)
		if !strings.Contains(got, want) {
			t.Fatalf("keyMoodDescription(A, %q) = %q, want substring %q", mode, got, want)
		}
	}
}

func TestReproduce(t *testing.T) {
	cfg := cliConfig{BPM: 122, Key: "Am", NoLLM: true, Bars: 16, Groove: "straight"}
	got := reproduceCmd(cfg, "12345")
	if !strings.Contains(got, "--bpm 122") || !strings.Contains(got, "--key Am") ||
		!strings.Contains(got, "--seed 12345") || !strings.Contains(got, "--no-llm") {
		t.Fatalf("unexpected reproduce cmd %q", got)
	}
	// straight groove and default bars should not appear
	if strings.Contains(got, "--groove") || strings.Contains(got, "--bars") {
		t.Fatalf("default values should be omitted from reproduce cmd %q", got)
	}

	// with non-default options
	cfg2 := cliConfig{BPM: 128, Key: "Dm-dorian", NoLLM: true, Bars: 32, Groove: "mpc60", OfflineStyle: "driving"}
	got2 := reproduceCmd(cfg2, "99")
	for _, want := range []string{"--bpm 128", "--key Dm-dorian", "--seed 99", "--no-llm", "--bars 32", "--groove mpc60", "--offline-style driving"} {
		if !strings.Contains(got2, want) {
			t.Fatalf("expected %q in reproduce cmd %q", want, got2)
		}
	}

	// AI provider
	cfg3 := cliConfig{BPM: 122, Key: "Am", NoLLM: false, ProviderName: "claude", Model: "claude-opus-4-7", Bars: 16, Groove: "straight"}
	got3 := reproduceCmd(cfg3, "42")
	if !strings.Contains(got3, "--provider claude") || !strings.Contains(got3, "--model claude-opus-4-7") {
		t.Fatalf("expected provider/model in reproduce cmd %q", got3)
	}
}

func TestCLIHelperFunctions(t *testing.T) {
	cases := map[float64]string{
		90:  "Downtempo / Ambient",
		110: "Deep House / Nu-Disco",
		120: "Tech House / Organic House",
		125: "Progressive House / Melodic Techno",
		135: "Peak Time Techno",
		145: "Hard Techno / Industrial",
	}
	for bpm, want := range cases {
		if got := bpmGenre(bpm); got != want {
			t.Fatalf("bpmGenre(%v) = %q, want %q", bpm, got, want)
		}
	}

	if got := modeLabel("major"); !strings.Contains(got, "Ionian") {
		t.Fatalf("unexpected major label %q", got)
	}
	if got := modeLabel("minor"); !strings.Contains(got, "Aeolian") {
		t.Fatalf("unexpected minor label %q", got)
	}
	if got := modeLabel("dorian"); !strings.Contains(got, "Dorian") {
		t.Fatalf("unexpected dorian label %q", got)
	}
	if got := scaleNoteString("A", "minor_natural"); !strings.Contains(got, "A") {
		t.Fatalf("unexpected scale string %q", got)
	}
	if got := scaleNoteString("?", "minor_natural"); got != "" {
		t.Fatalf("expected invalid scale string to be empty, got %q", got)
	}
}

func TestCLIFlowHelpers(t *testing.T) {
	testSelectModeOffline(t)
	testSelectModeQuit(t)
	testSelectLLMEngineFallback(t)
	testSelectLLMEngineOllama(t)
	testSelectLLMEngineOpenAI(t)
	testSelectTempo(t)
	testSelectKey(t)
}

func TestPrintSummaryAndBanner(t *testing.T) {
	cfg := cliConfig{BPM: 122, Key: "Am", OutputDir: "output", NoLLM: true}
	if out := captureStdout(t, func() { printSummary(cfg) }); !strings.Contains(out, "OUTPUT") {
		t.Fatalf("unexpected summary output %q", out)
	}

	tmp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := os.WriteFile("cadenzabanner.txt", []byte("BANNER"), 0o644); err != nil {
		t.Fatalf("write banner: %v", err)
	}
	if out := captureStdout(t, printBanner); !strings.Contains(out, "BANNER") {
		t.Fatalf("unexpected banner output %q", out)
	}
}

func TestRunInteractiveCLI_OfflineHappyPath(t *testing.T) {
	// flow: no quick start → skip preset → offline mode → skip energy → 122 BPM → Am → export-dir → confirm
	withInput("n\ns\n2\n\n122\nAm\nexport-dir\ny\n", func() {
		cfg, ok := runInteractiveCLI()
		if !ok {
			t.Fatal("expected interactive flow to complete")
		}
		if cfg.NoLLM != true || cfg.BPM != 122 || cfg.Key != "Am" || cfg.OutputDir != "export-dir" {
			t.Fatalf("unexpected cfg %+v", cfg)
		}
	})
}

func TestRunInteractiveCLI_WithEnergy(t *testing.T) {
	// flow: no quick start → skip preset → offline mode → energy 3 → 122 BPM → Am → export-dir → confirm
	withInput("n\ns\n2\n3\n122\nAm\nexport-dir\ny\n", func() {
		cfg, ok := runInteractiveCLI()
		if !ok {
			t.Fatal("expected interactive flow with energy to complete")
		}
		if cfg.Groove != "mpc60" {
			t.Fatalf("expected groove mpc60 from energy preset 3, got %q", cfg.Groove)
		}
		if cfg.OfflineStyle != "driving" {
			t.Fatalf("expected offline style driving from energy preset 3, got %q", cfg.OfflineStyle)
		}
	})
}

func TestRunInteractiveCLI_GenrePreset(t *testing.T) {
	// flow: no quick start → preset 2 (peak-time-driver) → export-dir → confirm
	withInput("n\n2\nexport-preset\ny\n", func() {
		cfg, ok := runInteractiveCLI()
		if !ok {
			t.Fatal("expected genre preset flow to complete")
		}
		if cfg.Key != "Am" || cfg.BPM != 130 {
			t.Fatalf("expected peak-time-driver preset (Am 130), got key=%q bpm=%.0f", cfg.Key, cfg.BPM)
		}
		if cfg.OfflineStyle != "driving" {
			t.Fatalf("expected driving style from peak-time-driver, got %q", cfg.OfflineStyle)
		}
	})
}

func TestRunInteractiveCLI_QuickStart(t *testing.T) {
	withInput("y\ny\n", func() {
		cfg, ok := runInteractiveCLI()
		if !ok {
			t.Fatal("expected quick start flow to complete")
		}
		if !cfg.NoLLM || cfg.BPM != 122 || cfg.Key != "Am" {
			t.Fatalf("unexpected quick start cfg %+v", cfg)
		}
		if !strings.Contains(cfg.OutputDir, "output") {
			t.Fatalf("expected timestamped output dir, got %q", cfg.OutputDir)
		}
	})
}

func TestProviderAndLoggerHelpers(t *testing.T) {
	appCfg := &config.AppConfig{}
	appCfg.LLM.Model = "claude-opus"

	if got := resolveDefaultModel("claude", appCfg); got != "claude-opus" {
		t.Fatalf("unexpected claude default %q", got)
	}
	if got := resolveDefaultModel("ollama", appCfg); got != "qwen2.5:7b" {
		t.Fatalf("unexpected ollama default %q", got)
	}
	if got := resolveDefaultModel("openai", appCfg); got != "gpt-4o" {
		t.Fatalf("unexpected openai default %q", got)
	}
	if got := resolveDefaultModel("gemini", appCfg); got != "gemini-2.0-flash-exp" {
		t.Fatalf("unexpected gemini default %q", got)
	}

	provider, err := buildProvider(true, "claude", "", "")
	if err != nil || provider.Name() != "mock" {
		t.Fatalf("unexpected offline provider result: %v / %v", provider, err)
	}

	ollama, err := buildProvider(false, "ollama", "qwen", "")
	if err != nil || !strings.Contains(ollama.Name(), "ollama/") {
		t.Fatalf("unexpected ollama provider result: %v / %v", ollama, err)
	}

	t.Setenv("OPENAI_API_KEY", "x")
	openai, err := buildProvider(false, "openai", "", "")
	if err != nil || openai.Name() != "openai" {
		t.Fatalf("unexpected openai provider result: %v / %v", openai, err)
	}

	t.Setenv("GEMINI_API_KEY", "x")
	gemini, err := buildProvider(false, "gemini", "", "")
	if err != nil || gemini.Name() != "gemini" {
		t.Fatalf("unexpected gemini provider result: %v / %v", gemini, err)
	}

	if parseSlogLevel("debug").String() != "DEBUG" || parseSlogLevel("warning").String() != "WARN" || parseSlogLevel("error").String() != "ERROR" {
		t.Fatal("unexpected slog level mapping")
	}

	outDir, err := os.MkdirTemp("", "cadenza-logger-test-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	cfg := &config.AppConfig{}
	cfg.Logging.Level = "debug"
	cfg.Logging.Format = "text"
	setupLogger(outDir, cfg)
	if _, err = os.Stat(filepath.Join(outDir, "cadenza.log")); err != nil {
		t.Fatalf("expected log file to exist: %v", err)
	}
}

func testSelectModeOffline(t *testing.T) {
	t.Helper()
	t.Run("select mode", func(t *testing.T) {
		var cfg cliConfig
		withInput("2\n", func() {
			assertOfflineModeSelection(t, &cfg, selectMode(&cfg))
		})
	})
}

func testSelectModeQuit(t *testing.T) {
	t.Helper()
	t.Run("select mode quit", func(t *testing.T) {
		var cfg cliConfig
		withInput("q\n", func() {
			if selectMode(&cfg) {
				t.Fatal("expected quit selection to return false")
			}
		})
	})
}

func testSelectLLMEngineFallback(t *testing.T) {
	t.Helper()
	t.Run("select llm engine fallback to offline", func(t *testing.T) {
		t.Setenv("ANTHROPIC_API_KEY", "")
		cfg := cliConfig{}
		withInput("1\ny\n", func() {
			assertClaudeOfflineFallback(t, &cfg, selectLLMEngine(&cfg))
		})
	})
}

func testSelectLLMEngineOllama(t *testing.T) {
	t.Helper()
	t.Run("select llm engine ollama", func(t *testing.T) {
		cfg := cliConfig{}
		withInput("2\nllama3\n", func() {
			if !selectLLMEngine(&cfg) {
				t.Fatal("expected ollama selection to succeed")
			}
		})
		if cfg.ProviderName != "ollama" || cfg.Model != "llama3" {
			t.Fatalf("unexpected ollama cfg %+v", cfg)
		}
	})
}

func testSelectLLMEngineOpenAI(t *testing.T) {
	t.Helper()
	t.Run("select llm engine openai", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "openai-test-key")
		cfg := cliConfig{}
		withInput("3\n\n", func() {
			if !selectLLMEngine(&cfg) {
				t.Fatal("expected openai selection to succeed")
			}
		})
		if cfg.ProviderName != "openai" || cfg.Model != "gpt-4o" {
			t.Fatalf("unexpected openai cfg %+v", cfg)
		}
	})
}

func testSelectTempo(t *testing.T) {
	t.Helper()
	t.Run("select tempo", func(t *testing.T) {
		var cfg cliConfig
		withInput("200\n123\n", func() {
			selectTempo(&cfg)
		})
		if cfg.BPM != 123 {
			t.Fatalf("expected bpm 123, got %v", cfg.BPM)
		}
	})
}

func testSelectKey(t *testing.T) {
	t.Helper()
	t.Run("select key", func(t *testing.T) {
		var cfg cliConfig
		withInput("bad\nAm\n", func() {
			selectKey(&cfg)
		})
		if cfg.Key != "Am" {
			t.Fatalf("expected key Am, got %q", cfg.Key)
		}
	})
	t.Run("select modal key", func(t *testing.T) {
		var cfg cliConfig
		out := captureStdout(t, func() {
			withInput("Dm-dorian\n", func() {
				selectKey(&cfg)
			})
		})
		if cfg.Key != "Dm-dorian" {
			t.Fatalf("expected modal key Dm-dorian, got %q", cfg.Key)
		}
		for _, want := range []string{"Dorian", "Phrygian", "Mixolydian", "Lydian"} {
			if !strings.Contains(out, want) {
				t.Fatalf("expected modal selector output to include %q, got %q", want, out)
			}
		}
	})
}

func TestConfirmInteractiveRender_Branches(t *testing.T) {
	t.Run("invalid then no", func(t *testing.T) {
		cfg := cliConfig{BPM: 122, Key: "Am"}
		withInput("bad\nn\n", func() {
			_, ok := confirmInteractiveRender(cfg)
			if ok {
				t.Fatal("expected cancel after invalid then no")
			}
		})
	})
}

func TestWantsQuickStart_InvalidThenNo(t *testing.T) {
	withInput("maybe\nn\n", func() {
		if wantsQuickStart() {
			t.Fatal("expected false after invalid then no")
		}
	})
}

func TestProviderHelpers(t *testing.T) {
	if got := providerEnvVar("gemini"); got != "GEMINI_API_KEY" {
		t.Fatalf("expected GEMINI_API_KEY, got %q", got)
	}
	if got := providerEnvVar("unknown"); got != "" {
		t.Fatalf("expected empty for unknown provider, got %q", got)
	}
	if got := providerSetupHint("openai"); !strings.Contains(got, "OPENAI_API_KEY") {
		t.Fatalf("expected openai hint, got %q", got)
	}
	if got := providerSetupHint("gemini"); !strings.Contains(got, "GEMINI_API_KEY") {
		t.Fatalf("expected gemini hint, got %q", got)
	}
	if got := providerSetupHint("unknown"); got != "" {
		t.Fatalf("expected empty for unknown provider hint, got %q", got)
	}
}

func TestKeyBPMHint(t *testing.T) {
	modes := []string{"minor", "major", "dorian", "phrygian", "mixolydian", "lydian", "unknown"}
	for _, mode := range modes {
		got := keyBPMHint(mode)
		if !strings.Contains(got, "BPM") {
			t.Fatalf("keyBPMHint(%q) missing BPM: %q", mode, got)
		}
	}
}

func TestModeLabel_AllModes(t *testing.T) {
	cases := map[string]string{
		"phrygian":   "Phrygian",
		"mixolydian": "Mixolydian",
		"lydian":     "Lydian",
		"unknown":    "unknown",
	}
	for mode, want := range cases {
		got := modeLabel(mode)
		if !strings.Contains(got, want) {
			t.Fatalf("modeLabel(%q) = %q, want substring %q", mode, got, want)
		}
	}
}

func TestSelectEnergy_InvalidInput(t *testing.T) {
	cfg := cliConfig{Groove: "straight"}
	withInput("bad\n6\n2\n", func() {
		selectEnergy(&cfg)
	})
	if cfg.OfflineStyle != "melodic" {
		t.Fatalf("expected energy 2 → melodic, got %q", cfg.OfflineStyle)
	}
}

func TestHandleProviderFailure(t *testing.T) {
	t.Run("retry same provider", func(t *testing.T) {
		cfg := cliConfig{ProviderName: "claude"}
		withInput("1\n", func() {
			if !handleProviderFailure(&cfg, fmt.Errorf("api key missing")) {
				t.Fatal("expected true (retry)")
			}
		})
		if cfg.NoLLM {
			t.Fatal("NoLLM must remain false for retry")
		}
	})

	t.Run("switch to offline", func(t *testing.T) {
		cfg := cliConfig{ProviderName: "claude"}
		withInput("2\n", func() {
			if !handleProviderFailure(&cfg, fmt.Errorf("api key missing")) {
				t.Fatal("expected true (switch offline)")
			}
		})
		if !cfg.NoLLM {
			t.Fatal("expected NoLLM=true after switching to offline")
		}
	})

	t.Run("cancel", func(t *testing.T) {
		cfg := cliConfig{ProviderName: "claude"}
		withInput("4\n", func() {
			if handleProviderFailure(&cfg, fmt.Errorf("api key missing")) {
				t.Fatal("expected false (cancel)")
			}
		})
	})

	t.Run("invalid then cancel", func(t *testing.T) {
		cfg := cliConfig{ProviderName: "claude"}
		withInput("bad\nq\n", func() {
			if handleProviderFailure(&cfg, fmt.Errorf("some error")) {
				t.Fatal("expected false after invalid then cancel")
			}
		})
	})

	t.Run("switch provider to ollama", func(t *testing.T) {
		cfg := cliConfig{ProviderName: "claude"}
		// choice 3 → selectLLMEngine → pick 2 (ollama) → default model (Enter)
		withInput("3\n2\nllama3\n", func() {
			if !handleProviderFailure(&cfg, fmt.Errorf("some error")) {
				t.Fatal("expected true after switching provider")
			}
		})
		if cfg.ProviderName != "ollama" {
			t.Fatalf("expected ollama after provider switch, got %q", cfg.ProviderName)
		}
	})
}

func TestRunInteractiveCLI_SetsInteractive(t *testing.T) {
	// flow: no quick start → skip preset → offline mode → skip energy → 122 BPM → Am → export-dir → confirm
	withInput("n\ns\n2\n\n122\nAm\nexport-dir2\ny\n", func() {
		cfg, ok := runInteractiveCLI()
		if !ok {
			t.Fatal("expected flow to complete")
		}
		if !cfg.Interactive {
			t.Fatal("expected Interactive=true from runInteractiveCLI")
		}
	})
}

func assertOfflineModeSelection(t *testing.T, cfg *cliConfig, ok bool) {
	t.Helper()
	if !ok || !cfg.NoLLM {
		t.Fatalf("expected offline mode selection, got ok=%v cfg=%+v", ok, *cfg)
	}
}

func assertClaudeOfflineFallback(t *testing.T, cfg *cliConfig, ok bool) {
	t.Helper()
	if !ok || !cfg.NoLLM || cfg.ProviderName != "claude" {
		t.Fatalf("unexpected cfg after fallback: ok=%v cfg=%+v", ok, *cfg)
	}
}
