package theory

import (
	"fmt"
	"testing"
)

func TestProgressionPool_Minor(t *testing.T) {
	pool := progressionPool("A", "minor_natural")
	if len(pool) == 0 {
		t.Fatal("expected non-empty pool for A minor")
	}
	for _, prog := range pool {
		if len(prog.Chords) != 4 {
			t.Errorf("progression should have 4 chords, got %d", len(prog.Chords))
		}
		if prog.Key != "A" || prog.Mode != "minor" {
			t.Errorf("expected key=A mode=minor, got key=%s mode=%s", prog.Key, prog.Mode)
		}
	}
}

func TestProgressionPool_Major(t *testing.T) {
	pool := progressionPool("C", "major")
	if len(pool) == 0 {
		t.Fatal("expected non-empty pool for C major")
	}
}

func TestSelectProgression_Deterministic(t *testing.T) {
	seed := "test-seed-123"
	prog1 := SelectProgression("A", "minor_natural", seed)
	prog2 := SelectProgression("A", "minor_natural", seed)
	if prog1.Chords[0].Root != prog2.Chords[0].Root {
		t.Error("same seed should produce same progression")
	}
}

func TestSelectProgression_DifferentSeeds(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		prog := SelectProgression("A", "minor_natural", fmt.Sprintf("seed-%d", i))
		key := prog.Chords[0].Root + prog.Chords[1].Root
		seen[key] = true
	}
	if len(seen) < 2 {
		t.Error("expected variation across different seeds")
	}
}
