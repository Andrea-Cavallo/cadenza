package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/generator"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

func TestValidateFlags(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(specPath, []byte("spec"), 0o644); err != nil {
		t.Fatalf("write spec file: %v", err)
	}

	if err := validateFlags(16, 1, "straight", "", 122, "Am"); err != nil {
		t.Fatalf("unexpected validateFlags error: %v", err)
	}
	if err := validateFlags(12, 1, "straight", "", 122, "Am"); err == nil {
		t.Fatal("expected bars validation error")
	}
	if err := validateFlags(16, 0, "straight", "", 122, "Am"); err == nil {
		t.Fatal("expected variations validation error")
	}
	if err := validateFlags(16, 1, "swing-nope", "", 122, "Am"); err == nil {
		t.Fatal("expected groove validation error")
	}
	if err := validateFlags(16, 1, "straight", specPath, 122, "Am"); err != nil {
		t.Fatalf("existing from-spec path should pass: %v", err)
	}
}

func TestParseCustomProgression_AndLabels(t *testing.T) {
	prog, err := parseCustomProgression("Am-F-C-G", "Am", 16)
	if err != nil {
		t.Fatalf("parseCustomProgression error: %v", err)
	}
	if len(prog.Chords) != 4 || prog.Chords[0].Bars != [2]int{1, 4} {
		t.Fatalf("unexpected progression: %+v", prog)
	}
	if _, err := parseCustomProgression("", "Am", 16); err == nil {
		t.Fatal("expected empty progression error")
	}
	if _, err := parseCustomProgression("Am", "Am", 16); err == nil {
		t.Fatal("expected too-short progression error")
	}
	if _, err := parseCustomProgression("Am-F-C", "Am", 16); err == nil {
		t.Fatal("expected uneven bar division error")
	}

	logged := progressionStringForLog(prog)
	if !strings.Contains(logged, "Am") || !strings.Contains(logged, "F") {
		t.Fatalf("unexpected progression log string %q", logged)
	}
	if trackLabel("foo_bassline.mid") != "[Bassline]" ||
		trackLabel("foo_arpeggio.mid") != "[Arpeggio]" ||
		trackLabel("foo_melody.mid") != "[Melody]" ||
		trackLabel("foo_other.mid") != "[Track]" {
		t.Fatal("track labels mismatch")
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()
	fn()
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func withOSStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	old := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	_ = w.Close()
	os.Stdin = r
	defer func() { os.Stdin = old }()
	fn()
}

func testCLIConfig(outputDir string) cliConfig {
	return cliConfig{
		BPM:          122,
		Key:          "Am",
		OutputDir:    outputDir,
		NoLLM:        true,
		ProviderName: "claude",
		Model:        "claude-opus-4-7",
		Bars:         16,
		Variations:   1,
		Groove:       "straight",
	}
}

func TestRunGenerationAndSingleGeneration(t *testing.T) {
	t.Run("invalid bpm and key", func(t *testing.T) {
		cfg := testCLIConfig(t.TempDir())
		cfg.BPM = 70
		errOut := captureStderr(t, func() { runGeneration(cfg) })
		if !strings.Contains(errOut, "BPM must be between 80 and 150") {
			t.Fatalf("unexpected bpm error output %q", errOut)
		}

		cfg = testCLIConfig(t.TempDir())
		cfg.Key = "bad"
		errOut = captureStderr(t, func() { runGeneration(cfg) })
		if !strings.Contains(errOut, "Invalid key") {
			t.Fatalf("unexpected key error output %q", errOut)
		}
	})

	t.Run("dry run and full offline generation", func(t *testing.T) {
		cfg := testCLIConfig(t.TempDir())
		cfg.DryRun = true
		out := captureStdout(t, func() { runGeneration(cfg) })
		if !strings.Contains(out, "DRY RUN") {
			t.Fatalf("unexpected dry run output %q", out)
		}

		cfg = testCLIConfig(t.TempDir())
		if err := runSingleGeneration(context.Background(), cfg, 1, "123"); err != nil {
			t.Fatalf("unexpected single generation error: %v", err)
		}
		files, err := filepath.Glob(filepath.Join(cfg.OutputDir, "*.mid"))
		if err != nil {
			t.Fatalf("glob midi files: %v", err)
		}
		if len(files) != 3 {
			t.Fatalf("expected 3 midi files, got %d", len(files))
		}
	})

	t.Run("custom progression and single-file/drums branches", func(t *testing.T) {
		cfg := testCLIConfig(t.TempDir())
		cfg.Progression = "Am-F-C-G"
		cfg.SingleFile = true
		cfg.Drums = true
		if err := runSingleGeneration(context.Background(), cfg, 2, "456"); err != nil {
			t.Fatalf("unexpected custom progression generation error: %v", err)
		}
	})
}

func TestRunFromSpecAndWatchMode(t *testing.T) {
	t.Run("from spec success and load failure", func(t *testing.T) {
		key := theory.Key{Root: "A", Mode: "minor", Scale: "minor_natural"}
		ctx := generator.MusicContext{
			BPM:              122,
			Key:              key,
			Bars:             16,
			VariationSeed:    "seed-z",
			ChordProgression: theory.SelectProgression("A", "minor_natural", "seed-z"),
			Groove:           "straight",
		}
		spec := generator.OfflineTemplate("bassline", ctx)
		specPath := filepath.Join(t.TempDir(), "spec.yaml")
		if err := schema.DumpToYAML(spec, specPath); err != nil {
			t.Fatalf("dump yaml: %v", err)
		}

		cfg := testCLIConfig(t.TempDir())
		cfg.FromSpec = specPath
		out := captureStdout(t, func() { runFromSpec(cfg) })
		if !strings.Contains(out, "Re-rendered") {
			t.Fatalf("unexpected runFromSpec output %q", out)
		}

		cfg.FromSpec = filepath.Join(t.TempDir(), "missing.yaml")
		out = captureStdout(t, func() { runFromSpec(cfg) })
		if !strings.Contains(out, "Failed to load spec") {
			t.Fatalf("unexpected missing spec output %q", out)
		}
	})

	t.Run("watch mode quit and one generation", func(t *testing.T) {
		cfg := testCLIConfig(t.TempDir())
		seed := uint64(100)
		cfg.Seed = &seed
		out := captureStdout(t, func() {
			withOSStdin(t, "q\n", func() {
				runWatchMode(cfg)
			})
		})
		if !strings.Contains(out, "Watch mode exited") {
			t.Fatalf("unexpected watch quit output %q", out)
		}

		cfg = testCLIConfig(t.TempDir())
		cfg.Seed = &seed
		out = captureStdout(t, func() {
			withOSStdin(t, "\nq\n", func() {
				runWatchMode(cfg)
			})
		})
		if !strings.Contains(out, "Variation 1 complete") {
			t.Fatalf("unexpected watch generation output %q", out)
		}
	})
}
