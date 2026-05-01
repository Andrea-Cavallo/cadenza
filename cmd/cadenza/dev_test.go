package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/cache"
	"github.com/Andrea-Cavallo/cadenza/internal/generator"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

func captureStdoutDev(t *testing.T, fn func()) string {
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

func TestDevFlagParsingHelpers(t *testing.T) {
	cfg := parseGenerateFlags([]string{"generate", "--bpm", "126", "--key", "Dm", "--no-llm", "--provider", "ollama", "--seed", "42"})
	if cfg.BPM != 126 || cfg.Key != "Dm" || !cfg.NoLLM || cfg.ProviderName != "ollama" || cfg.Seed == nil || *cfg.Seed != 42 {
		t.Fatalf("unexpected parsed generate cfg %+v", cfg)
	}

	args := []string{"cmd", "--foo", "bar"}
	idx := 1
	if got, ok := flagValue(args, &idx); !ok || got != "bar" {
		t.Fatalf("unexpected flagValue result %q %v", got, ok)
	}
	idx = len(args) - 1
	if _, ok := flagValue(args, &idx); ok {
		t.Fatal("expected missing flag value to fail")
	}

	val := 100.0
	setFloatFlag("123.5", &val)
	if val != 123.5 {
		t.Fatalf("expected updated float value, got %v", val)
	}
	setFloatFlag("oops", &val)
	if val != 123.5 {
		t.Fatalf("invalid float should not change value, got %v", val)
	}

	if got := parseUintPtr("55"); got == nil || *got != 55 {
		t.Fatalf("unexpected uint parse result %v", got)
	}
	if got := parseUintPtr("bad"); got != nil {
		t.Fatalf("expected nil invalid uint parse, got %v", got)
	}
}

func TestDevCommandHelpers(t *testing.T) {
	key, seed, custom := parseChordProgressionFlags([]string{"cp", "--key", "Dm", "--seed", "99", "--custom", "Dm-Bb-F-C"})
	if key != "Dm" || seed != "99" || custom != "Dm-Bb-F-C" {
		t.Fatalf("unexpected chord flags %q %q %q", key, seed, custom)
	}

	if out := captureStdoutDev(t, printDevHelp); !strings.Contains(out, "AVAILABLE COMMANDS") {
		t.Fatalf("unexpected help output %q", out)
	}
	if out := captureStdoutDev(t, func() { devRender(nil, [3]*schema.PatternSpec{}) }); !strings.Contains(out, "not implemented yet") {
		t.Fatalf("unexpected devRender output %q", out)
	}
	if out := captureStdoutDev(t, func() { devInspect(nil, [3]*schema.PatternSpec{}) }); !strings.Contains(out, "not implemented yet") {
		t.Fatalf("unexpected devInspect output %q", out)
	}
	if out := captureStdoutDev(t, func() { devFromSpec(nil, nil) }); !strings.Contains(out, "not implemented yet") {
		t.Fatalf("unexpected devFromSpec output %q", out)
	}
	if out := captureStdoutDev(t, func() { devDumpSpec(nil, [3]*schema.PatternSpec{}) }); !strings.Contains(out, "not implemented yet") {
		t.Fatalf("unexpected devDumpSpec output %q", out)
	}
}

func TestDevCacheInfoAndChordProgression(t *testing.T) {
	if out := captureStdoutDev(t, func() { devCacheInfo(nil) }); !strings.Contains(out, "Cache not initialized") {
		t.Fatalf("unexpected nil cache output %q", out)
	}

	c := cache.New(30, t.TempDir())
	_ = c.Set([]byte(`"x"`), "a")
	_, _ = c.Get("missing")
	if out := captureStdoutDev(t, func() { devCacheInfo(c) }); !strings.Contains(out, "CACHE STATISTICS") {
		t.Fatalf("unexpected cache stats output %q", out)
	}

	if prog := devChordProgression([]string{"cp", "--key", "Am"}); len(prog.Chords) == 0 {
		t.Fatal("expected generated progression")
	}
	if prog := devChordProgression([]string{"cp", "--key", "bad"}); len(prog.Chords) != 0 {
		t.Fatal("expected invalid key to return empty progression")
	}
	if prog := devChordProgression([]string{"cp", "--key", "Am", "--custom", "Am-F-C-G"}); len(prog.Chords) != 4 {
		t.Fatalf("expected custom progression, got %+v", prog)
	}
}

func TestRunDevMode_QuitAndCommands(t *testing.T) {
	t.Run("quit immediately", func(t *testing.T) {
		out := captureStdoutDev(t, func() {
			withOSStdin(t, "q\n", func() { runDevMode() })
		})
		if !strings.Contains(out, "Dev mode exited") {
			t.Fatalf("expected dev mode exit output, got %q", out)
		}
	})

	t.Run("help and unknown command then quit", func(t *testing.T) {
		out := captureStdoutDev(t, func() {
			withOSStdin(t, "help\nunknown-xyz\nq\n", func() { runDevMode() })
		})
		if !strings.Contains(out, "AVAILABLE COMMANDS") {
			t.Fatalf("expected help output, got %q", out)
		}
		if !strings.Contains(out, "Unknown command") {
			t.Fatalf("expected unknown command message, got %q", out)
		}
	})

	t.Run("generate offline via dev mode then quit", func(t *testing.T) {
		out := captureStdoutDev(t, func() {
			withOSStdin(t, "generate --bpm 122 --key Am --no-llm\nq\n", func() { runDevMode() })
		})
		if !strings.Contains(out, "Generated 3 patterns") {
			t.Fatalf("expected generation success, got %q", out)
		}
	})

	t.Run("all not-yet-implemented commands then quit", func(t *testing.T) {
		out := captureStdoutDev(t, func() {
			withOSStdin(t, "render\ninspect\nfrom-spec\ndump-spec\ncp --key Am\ncache\nq\n", func() { runDevMode() })
		})
		if !strings.Contains(out, "not implemented yet") {
			t.Fatalf("expected not-implemented message for render/inspect, got %q", out)
		}
		if !strings.Contains(out, "Chord Progression") {
			t.Fatalf("expected chord progression output from cp command, got %q", out)
		}
	})
}

func TestDevGenerate(t *testing.T) {
	t.Run("invalid BPM", func(t *testing.T) {
		out := captureStdoutDev(t, func() {
			devGenerate(context.Background(), []string{"generate", "--bpm", "50", "--no-llm"}, nil)
		})
		if !strings.Contains(out, "BPM must be 80-150") {
			t.Fatalf("expected BPM error, got %q", out)
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		out := captureStdoutDev(t, func() {
			devGenerate(context.Background(), []string{"generate", "--bpm", "122", "--key", "zzz", "--no-llm"}, nil)
		})
		if !strings.Contains(out, "Invalid key") {
			t.Fatalf("expected invalid key error, got %q", out)
		}
	})

	t.Run("valid offline generation", func(t *testing.T) {
		out := captureStdoutDev(t, func() {
			devGenerate(context.Background(), []string{"generate", "--bpm", "122", "--key", "Am", "--no-llm"}, nil)
		})
		if !strings.Contains(out, "Generated 3 patterns") {
			t.Fatalf("expected generation success, got %q", out)
		}
	})
}

func TestDevValidate(t *testing.T) {
	t.Run("no spec flag", func(t *testing.T) {
		out := captureStdoutDev(t, func() {
			devValidate([]string{"validate"}, theory.ChordProgression{})
		})
		if !strings.Contains(out, "Usage: validate --spec") {
			t.Fatalf("expected usage message, got %q", out)
		}
	})

	t.Run("nonexistent spec file", func(t *testing.T) {
		out := captureStdoutDev(t, func() {
			devValidate([]string{"validate", "--spec", "/nonexistent/path.yaml"}, theory.ChordProgression{})
		})
		if !strings.Contains(out, "Load failed") {
			t.Fatalf("expected load failed message, got %q", out)
		}
	})

	t.Run("valid spec file passes validation", func(t *testing.T) {
		key := theory.Key{Root: "A", Mode: "minor", Scale: "minor_natural"}
		prog := theory.SelectProgression("A", "minor_natural", "test-seed")
		ctx := generator.MusicContext{BPM: 122, Key: key, Bars: 16, VariationSeed: "test-seed", ChordProgression: prog}
		spec := generator.OfflineTemplate("bassline", ctx)
		specPath := filepath.Join(t.TempDir(), "spec.yaml")
		if err := schema.DumpToYAML(spec, specPath); err != nil {
			t.Fatalf("dump yaml: %v", err)
		}
		out := captureStdoutDev(t, func() {
			devValidate([]string{"validate", "--spec", specPath}, prog)
		})
		if !strings.Contains(out, "Validation passed") {
			t.Fatalf("expected validation passed, got %q", out)
		}
	})
}
