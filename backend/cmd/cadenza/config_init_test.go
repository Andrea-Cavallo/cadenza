package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunConfigInit(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "cadenza.yaml")

	out := captureStdout(t, func() {
		if err := runConfigInit(path, false); err != nil {
			t.Fatalf("runConfigInit error: %v", err)
		}
	})
	if !strings.Contains(out, "Created") {
		t.Fatalf("expected created message, got %q", out)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	for _, want := range []string{"audio:", "llm:", "output:", "cache:"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("starter config missing %q:\n%s", want, string(raw))
		}
	}

	if err := runConfigInit(path, false); err == nil {
		t.Fatal("expected existing config to fail without force")
	}
	if err := runConfigInit(path, true); err != nil {
		t.Fatalf("force overwrite should pass: %v", err)
	}
}

func TestHandleConfigCommandNotConfig(t *testing.T) {
	if handleConfigCommand([]string{"--bpm", "122"}) {
		t.Fatal("expected non-config args to be ignored")
	}
}

func TestHandleConfigCommandInit(t *testing.T) {
	t.Chdir(t.TempDir())
	out := captureStdout(t, func() {
		if !handleConfigCommand([]string{"config", "init", "--force"}) {
			t.Fatal("expected config init to be handled")
		}
	})
	if !strings.Contains(out, "Created") {
		t.Fatalf("expected created output, got %q", out)
	}
	if _, err := os.Stat(defaultConfigPath); err != nil {
		t.Fatalf("expected default config to exist: %v", err)
	}
}

func TestRunConfigInitWriteError(t *testing.T) {
	if err := runConfigInit(t.TempDir(), true); err == nil {
		t.Fatal("expected writing to a directory path to fail")
	}
}

func TestParseConfigInitArgs(t *testing.T) {
	handled, force, err := parseConfigInitArgs([]string{"--bpm", "122"})
	if handled || force || err != nil {
		t.Fatalf("expected unrelated args to be ignored, got handled=%v force=%v err=%v", handled, force, err)
	}

	handled, force, err = parseConfigInitArgs([]string{"config", "init", "--force"})
	if !handled || !force || err != nil {
		t.Fatalf("expected forced init, got handled=%v force=%v err=%v", handled, force, err)
	}

	handled, force, err = parseConfigInitArgs([]string{"config", "show"})
	if !handled || force || err == nil {
		t.Fatalf("expected usage error, got handled=%v force=%v err=%v", handled, force, err)
	}

	handled, force, err = parseConfigInitArgs([]string{"config", "init", "--bad"})
	if !handled || force || err == nil {
		t.Fatalf("expected unknown option error, got handled=%v force=%v err=%v", handled, force, err)
	}
}
