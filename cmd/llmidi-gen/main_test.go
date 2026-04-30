package main

import "testing"

func TestTrackLabel(t *testing.T) {
	if trackLabel("foo_bassline.mid") != "[Bassline]" {
		t.Fatal("bassline label mismatch")
	}
	if trackLabel("foo_arpeggio.mid") != "[Arpeggio]" {
		t.Fatal("arpeggio label mismatch")
	}
	if trackLabel("foo_melody.mid") != "[Melody]" {
		t.Fatal("melody label mismatch")
	}
	if trackLabel("foo.mid") != "[Track]" {
		t.Fatal("default label mismatch")
	}
}

func TestBuildProvider(t *testing.T) {
	provider, err := buildProvider(true, "claude", "")
	if err != nil {
		t.Fatalf("offline buildProvider error: %v", err)
	}
	if provider.Name() != "mock" {
		t.Fatalf("expected mock provider, got %q", provider.Name())
	}

	if _, err := buildProvider(false, "nope", ""); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}
