package main

import (
	"bufio"
	"bytes"
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
	if got := modeLabel("dorian"); got != "dorian" {
		t.Fatalf("unexpected passthrough label %q", got)
	}
	if got := scaleNoteString("A", "minor_natural"); !strings.Contains(got, "A") {
		t.Fatalf("unexpected scale string %q", got)
	}
	if got := scaleNoteString("?", "minor_natural"); got != "" {
		t.Fatalf("expected invalid scale string to be empty, got %q", got)
	}
}

func TestCLIFlowHelpers(t *testing.T) {
	t.Run("select mode", func(t *testing.T) {
		var cfg cliConfig
		withInput("2\n", func() {
			if ok := selectMode(&cfg); !ok || !cfg.NoLLM {
				t.Fatalf("expected offline mode selection, got ok=%v cfg=%+v", ok, cfg)
			}
		})
	})

	t.Run("select mode quit", func(t *testing.T) {
		var cfg cliConfig
		withInput("q\n", func() {
			if ok := selectMode(&cfg); ok {
				t.Fatal("expected quit selection to return false")
			}
		})
	})

	t.Run("select llm engine fallback to offline", func(t *testing.T) {
		t.Setenv("ANTHROPIC_API_KEY", "")
		cfg := cliConfig{}
		withInput("1\ny\n", func() {
			if ok := selectLLMEngine(&cfg); !ok || !cfg.NoLLM || cfg.ProviderName != "claude" {
				t.Fatalf("unexpected cfg after fallback: ok=%v cfg=%+v", ok, cfg)
			}
		})
	})

	t.Run("select llm engine ollama", func(t *testing.T) {
		cfg := cliConfig{}
		withInput("2\nllama3\n", func() {
			if ok := selectLLMEngine(&cfg); !ok {
				t.Fatal("expected ollama selection to succeed")
			}
		})
		if cfg.ProviderName != "ollama" || cfg.Model != "llama3" {
			t.Fatalf("unexpected ollama cfg %+v", cfg)
		}
	})

	t.Run("select tempo", func(t *testing.T) {
		var cfg cliConfig
		withInput("200\n123\n", func() {
			selectTempo(&cfg)
		})
		if cfg.BPM != 123 {
			t.Fatalf("expected bpm 123, got %v", cfg.BPM)
		}
	})

	t.Run("select key", func(t *testing.T) {
		var cfg cliConfig
		withInput("bad\nAm\n", func() {
			selectKey(&cfg)
		})
		if cfg.Key != "Am" {
			t.Fatalf("expected key Am, got %q", cfg.Key)
		}
	})
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
	withInput("2\n122\nAm\nexport-dir\ny\n", func() {
		cfg, ok := runInteractiveCLI()
		if !ok {
			t.Fatal("expected interactive flow to complete")
		}
		if cfg.NoLLM != true || cfg.BPM != 122 || cfg.Key != "Am" || cfg.OutputDir != "export-dir" {
			t.Fatalf("unexpected cfg %+v", cfg)
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
	if _, err := os.Stat(filepath.Join(outDir, "cadenza.log")); err != nil {
		t.Fatalf("expected log file to exist: %v", err)
	}
}
