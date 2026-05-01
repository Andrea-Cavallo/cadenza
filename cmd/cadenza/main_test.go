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

	if err := validateFlags(16, 1, "straight", "", "", 122, "Am"); err != nil {
		t.Fatalf("unexpected validateFlags error: %v", err)
	}
	if err := validateFlags(12, 1, "straight", "", "", 122, "Am"); err == nil {
		t.Fatal("expected bars validation error")
	}
	if err := validateFlags(16, 0, "straight", "", "", 122, "Am"); err == nil {
		t.Fatal("expected variations validation error")
	}
	if err := validateFlags(16, 1, "swing-nope", "", "", 122, "Am"); err == nil {
		t.Fatal("expected groove validation error")
	}
	if err := validateFlags(16, 1, "straight", "", "bad-style", 122, "Am"); err == nil {
		t.Fatal("expected offline-style validation error")
	}
	if err := validateFlags(16, 1, "straight", "", "hypnotic", 122, "Am"); err != nil {
		t.Fatalf("valid offline-style should pass: %v", err)
	}
	if err := validateFlags(16, 1, "straight", specPath, "", 122, "Am"); err != nil {
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

func TestHandlePostRunAction(t *testing.T) {
	t.Run("exit", func(t *testing.T) {
		cfg := testCLIConfig(t.TempDir())
		withInput("q\n", func() {
			if handlePostRunAction(cfg) {
				t.Fatal("expected post-run exit")
			}
		})
	})

	t.Run("same setup new seed then exit", func(t *testing.T) {
		cfg := testCLIConfig(t.TempDir())
		withInput("2\nq\n", func() {
			if handlePostRunAction(cfg) {
				t.Fatal("expected post-run exit after rerun")
			}
		})
		files, err := filepath.Glob(filepath.Join(cfg.OutputDir, "*.mid"))
		if err != nil {
			t.Fatalf("glob midi files: %v", err)
		}
		if len(files) != 3 {
			t.Fatalf("expected 3 midi files after same-setup rerun, got %d", len(files))
		}
	})

	t.Run("same harmony new motifs then exit", func(t *testing.T) {
		cfg := testCLIConfig(t.TempDir())
		lastRun = lastRunInfo{Seed: "123", ProgCLI: "Am-F-C-G", BPM: 122, Key: "Am"}
		withInput("3\nq\n", func() {
			if handlePostRunAction(cfg) {
				t.Fatal("expected post-run exit after same-harmony")
			}
		})
		files, _ := filepath.Glob(filepath.Join(cfg.OutputDir, "*.mid"))
		if len(files) != 3 {
			t.Fatalf("expected 3 midi files after same-harmony, got %d", len(files))
		}
	})

	t.Run("faster then exit", func(t *testing.T) {
		cfg := testCLIConfig(t.TempDir())
		cfg.BPM = 122
		withInput("5\nq\n", func() {
			if handlePostRunAction(cfg) {
				t.Fatal("expected post-run exit after faster")
			}
		})
	})

	t.Run("slower then exit", func(t *testing.T) {
		cfg := testCLIConfig(t.TempDir())
		cfg.BPM = 122
		withInput("6\nq\n", func() {
			if handlePostRunAction(cfg) {
				t.Fatal("expected post-run exit after slower")
			}
		})
	})

	t.Run("busier then exit", func(t *testing.T) {
		cfg := testCLIConfig(t.TempDir())
		withInput("7\nq\n", func() {
			if handlePostRunAction(cfg) {
				t.Fatal("expected post-run exit after busier")
			}
		})
	})

	t.Run("sparser then exit", func(t *testing.T) {
		cfg := testCLIConfig(t.TempDir())
		withInput("8\nq\n", func() {
			if handlePostRunAction(cfg) {
				t.Fatal("expected post-run exit after sparser")
			}
		})
	})

	t.Run("AB compare then exit", func(t *testing.T) {
		outDir := t.TempDir()
		cfg := testCLIConfig(outDir)
		withInput("4\nq\n", func() {
			if handlePostRunAction(cfg) {
				t.Fatal("expected post-run exit after AB compare")
			}
		})
		filesA, _ := filepath.Glob(filepath.Join(outDir, "A", "*.mid"))
		filesB, _ := filepath.Glob(filepath.Join(outDir, "B", "*.mid"))
		if len(filesA) != 3 || len(filesB) != 3 {
			t.Fatalf("expected 3 files in A and 3 in B, got A=%d B=%d", len(filesA), len(filesB))
		}
	})

	t.Run("invalid then exit", func(t *testing.T) {
		cfg := testCLIConfig(t.TempDir())
		withInput("bad\nq\n", func() {
			if handlePostRunAction(cfg) {
				t.Fatal("expected post-run exit after invalid")
			}
		})
	})
}

func TestProgressionToCLIString(t *testing.T) {
	prog, err := parseCustomProgression("Am-F-C-G", "Am", 16)
	if err != nil {
		t.Fatalf("parseCustomProgression: %v", err)
	}
	got := progressionToCLIString(prog)
	if got != "Am-F-C-G" {
		t.Fatalf("expected Am-F-C-G, got %q", got)
	}
}

func TestClampBPM(t *testing.T) {
	if clampBPM(50) != 80 {
		t.Fatal("expected 80 for clamp below min")
	}
	if clampBPM(200) != 150 {
		t.Fatal("expected 150 for clamp above max")
	}
	if clampBPM(122) != 122 {
		t.Fatal("expected 122 unchanged")
	}
}

func TestApplyGenrePreset(t *testing.T) {
	cfg := cliConfig{Groove: "straight"}
	if !applyGenrePreset(&cfg, "progressive-warmup") {
		t.Fatal("expected progressive-warmup to apply")
	}
	if cfg.Key != "Am-dorian" || cfg.BPM != 122 || cfg.OfflineStyle != "melodic" {
		t.Fatalf("unexpected preset cfg %+v", cfg)
	}
	if applyGenrePreset(&cfg, "unknown-preset") {
		t.Fatal("expected unknown preset to return false")
	}
}
