package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
