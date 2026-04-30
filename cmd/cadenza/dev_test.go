package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/cache"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
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
