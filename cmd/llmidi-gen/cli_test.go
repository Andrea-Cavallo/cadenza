package main

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"testing"
)

func withLLMIDIInput(input string, fn func()) {
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader(input))
	defer func() { stdinReader = old }()
	fn()
}

func captureLLMIDIStdout(t *testing.T, fn func()) string {
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

func TestLLMIDICLIHelpers(t *testing.T) {
	withLLMIDIInput("hello\n", func() {
		if got := ask("Label"); got != "hello" {
			t.Fatalf("unexpected ask result %q", got)
		}
	})
	withLLMIDIInput("\n", func() {
		if got := askDefault("Label", "fallback"); got != "fallback" {
			t.Fatalf("unexpected askDefault fallback %q", got)
		}
	})

	if bpmGenre(90) == "" || bpmGenre(145) == "" {
		t.Fatal("expected bpm genres")
	}
	if modeLabel("major") == "major" || modeLabel("minor") == "minor" {
		t.Fatal("expected friendly mode labels")
	}
	if got := scaleNoteString("A", "minor_natural"); !strings.Contains(got, "A") {
		t.Fatalf("unexpected scale string %q", got)
	}
	if got := scaleNoteString("?", "minor_natural"); got != "" {
		t.Fatalf("expected invalid scale string to be empty, got %q", got)
	}
}

func TestLLMIDICLIFlow(t *testing.T) {
	t.Run("select mode offline", func(t *testing.T) {
		var cfg cliConfig
		withLLMIDIInput("2\n", func() {
			if ok := selectMode(&cfg); !ok || !cfg.NoLLM {
				t.Fatalf("unexpected mode result ok=%v cfg=%+v", ok, cfg)
			}
		})
	})

	t.Run("select engine ollama", func(t *testing.T) {
		cfg := cliConfig{}
		withLLMIDIInput("2\nmymodel\n", func() {
			if ok := selectLLMEngine(&cfg); !ok {
				t.Fatal("expected engine selection to pass")
			}
		})
		if cfg.ProviderName != "ollama" || cfg.Model != "mymodel" {
			t.Fatalf("unexpected engine cfg %+v", cfg)
		}
	})

	t.Run("tempo key and interactive run", func(t *testing.T) {
		withLLMIDIInput("2\n122\nAm\nout\ny\n", func() {
			cfg, ok := runInteractiveCLI()
			if !ok {
				t.Fatal("expected interactive flow ok")
			}
			if !cfg.NoLLM || cfg.BPM != 122 || cfg.Key != "Am" || cfg.OutputDir != "out" {
				t.Fatalf("unexpected cfg %+v", cfg)
			}
		})
	})
}

func TestLLMIDIBannerAndSummary(t *testing.T) {
	if out := captureLLMIDIStdout(t, printBanner); !strings.Contains(out, "L L M I D I - G E N") {
		t.Fatalf("unexpected banner output %q", out)
	}
	if out := captureLLMIDIStdout(t, func() {
		printSummary(cliConfig{BPM: 122, Key: "Am", NoLLM: true, OutputDir: "output"})
	}); !strings.Contains(out, "OUTPUT") {
		t.Fatalf("unexpected summary output %q", out)
	}
}
