package theory

import "testing"

func TestScaleNotes(t *testing.T) {
	tests := []struct {
		root     string
		scaleTyp string
		want     []string
	}{
		{"A", "minor_natural", []string{"A", "B", "C", "D", "E", "F", "G"}},
		{"C", "major", []string{"C", "D", "E", "F", "G", "A", "B"}},
		{"D", "major", []string{"D", "E", "F#", "G", "A", "B", "C#"}},
		{"E", "minor_natural", []string{"E", "F#", "G", "A", "B", "C", "D"}},
	}
	for _, tt := range tests {
		t.Run(tt.root+"_"+tt.scaleTyp, func(t *testing.T) {
			got, err := ScaleNotes(tt.root, tt.scaleTyp)
			if err != nil {
				t.Fatalf("ScaleNotes(%q, %q) error: %v", tt.root, tt.scaleTyp, err)
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

func TestNoteInScale(t *testing.T) {
	tests := []struct {
		note     string
		root     string
		scaleTyp string
		want     bool
	}{
		{"A", "A", "minor_natural", true},
		{"C", "A", "minor_natural", true},
		{"C#", "A", "minor_natural", false},
		{"F#", "D", "major", true},
		{"Bb", "D", "major", false},
	}
	for _, tt := range tests {
		name := tt.note + "_in_" + tt.root + "_" + tt.scaleTyp
		t.Run(name, func(t *testing.T) {
			got, err := NoteInScale(tt.note, tt.root, tt.scaleTyp)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if got != tt.want {
				t.Errorf("NoteInScale(%q, %q, %q) = %v, want %v",
					tt.note, tt.root, tt.scaleTyp, got, tt.want)
			}
		})
	}
}
