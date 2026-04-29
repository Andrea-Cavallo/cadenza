package theory

import "testing"

func TestNoteToMIDI(t *testing.T) {
	tests := []struct {
		note string
		midi int
	}{
		{"C4", 60},
		{"A4", 69},
		{"A2", 45},
		{"A1", 33},
		{"G3", 55},
		{"C3", 48},
		{"C6", 84},
		{"C7", 96},
		{"D#3", 51},
		{"Bb4", 70},
		{"F#2", 42},
	}
	for _, tt := range tests {
		t.Run(tt.note, func(t *testing.T) {
			got, err := NoteToMIDI(tt.note)
			if err != nil {
				t.Fatalf("NoteToMIDI(%q) error: %v", tt.note, err)
			}
			if got != tt.midi {
				t.Errorf("NoteToMIDI(%q) = %d, want %d", tt.note, got, tt.midi)
			}
		})
	}
}

func TestMIDIToNote(t *testing.T) {
	tests := []struct {
		midi int
		note string
	}{
		{60, "C4"},
		{69, "A4"},
		{45, "A2"},
		{33, "A1"},
	}
	for _, tt := range tests {
		t.Run(tt.note, func(t *testing.T) {
			got := MIDIToNote(tt.midi)
			if got != tt.note {
				t.Errorf("MIDIToNote(%d) = %q, want %q", tt.midi, got, tt.note)
			}
		})
	}
}

func TestNoteToMIDI_Invalid(t *testing.T) {
	invalids := []string{"", "X4", "C", "C-1", "H3"}
	for _, note := range invalids {
		t.Run(note, func(t *testing.T) {
			_, err := NoteToMIDI(note)
			if err == nil {
				t.Errorf("NoteToMIDI(%q) should return error", note)
			}
		})
	}
}
