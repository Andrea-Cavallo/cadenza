package theory

import "testing"

func TestChordNotes(t *testing.T) {
	tests := []struct {
		root    string
		quality string
		want    []string
	}{
		{"A", "minor", []string{"A", "C", "E"}},
		{"C", "major", []string{"C", "E", "G"}},
		{"D", "minor", []string{"D", "F", "A"}},
		{"F", "major", []string{"F", "A", "C"}},
		{"G", "major", []string{"G", "B", "D"}},
		{"E", "minor", []string{"E", "G", "B"}},
		{"B", "dim", []string{"B", "D", "F"}},
	}
	for _, tt := range tests {
		t.Run(tt.root+"_"+tt.quality, func(t *testing.T) {
			got, err := ChordNotes(tt.root, tt.quality)
			if err != nil {
				t.Fatalf("ChordNotes(%q, %q) error: %v", tt.root, tt.quality, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d notes, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("note[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestChordsInKey(t *testing.T) {
	chords, err := ChordsInKey("A", "minor_natural")
	if err != nil {
		t.Fatalf("ChordsInKey error: %v", err)
	}
	if len(chords) != 7 {
		t.Fatalf("expected 7 diatonic chords, got %d", len(chords))
	}
	wantRoots := []string{"A", "B", "C", "D", "E", "F", "G"}
	wantQualities := []string{"minor", "dim", "major", "minor", "minor", "major", "major"}
	for i, c := range chords {
		if c.Root != wantRoots[i] {
			t.Errorf("chord[%d].Root = %q, want %q", i, c.Root, wantRoots[i])
		}
		if c.Quality != wantQualities[i] {
			t.Errorf("chord[%d].Quality = %q, want %q", i, c.Quality, wantQualities[i])
		}
	}
}
