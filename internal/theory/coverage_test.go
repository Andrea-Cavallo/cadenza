package theory

import "testing"

func TestParseChordName_AndQualitySuffix(t *testing.T) {
	tests := []struct {
		input       string
		wantRoot    string
		wantQuality string
		wantSuffix  string
		expectErr   bool
	}{
		{"C", "C", "major", "", false},
		{"Am", "A", "minor", "m", false},
		{"Bdim", "B", "dim", "dim", false},
		{"Faug", "F", "aug", "aug", false},
		{"Dsus2", "D", "sus2", "", false},
		{"Esus4", "E", "sus4", "", false},
		{"H7", "", "", "", true},
	}

	for _, tt := range tests {
		root, quality, err := ParseChordName(tt.input)
		if tt.expectErr {
			if err == nil {
				t.Fatalf("%s: expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: unexpected error %v", tt.input, err)
		}
		if root != tt.wantRoot || quality != tt.wantQuality {
			t.Fatalf("%s: got root=%s quality=%s", tt.input, root, quality)
		}
		if got := QualitySuffix(quality); got != tt.wantSuffix {
			t.Fatalf("%s: suffix=%q want %q", tt.input, got, tt.wantSuffix)
		}
	}
}

func TestNoteNameOnly_AndScaleHelpers(t *testing.T) {
	if got := NoteNameOnly("A#3"); got != "A#" {
		t.Fatalf("A#3 => %q", got)
	}
	if got := NoteNameOnly("Bb4"); got != "Bb" {
		t.Fatalf("Bb4 => %q", got)
	}
	if got := NoteNameOnly("C"); got != "C" {
		t.Fatalf("C => %q", got)
	}
	if got := NoteNameOnly(""); got != "" {
		t.Fatalf("empty => %q", got)
	}

	if _, err := rootSemitone("H"); err == nil {
		t.Fatal("expected invalid rootSemitone error")
	}
	if notes, err := ScaleNotes("Bb", "major"); err != nil || len(notes) != 7 || notes[1] != "C" {
		t.Fatalf("unexpected Bb major notes: %v %v", notes, err)
	}
	if _, err := ScaleNotes("C", "lydian-nope"); err == nil {
		t.Fatal("expected unknown scale type error")
	}
}
