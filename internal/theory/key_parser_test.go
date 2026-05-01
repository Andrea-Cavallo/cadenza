package theory

import "testing"

func TestParseKey(t *testing.T) {
	tests := []struct {
		input string
		root  string
		mode  string
		scale string
	}{
		{"Am", "A", "minor", "minor_natural"},
		{"D", "D", "major", "major"},
		{"F#m", "F#", "minor", "minor_natural"},
		{"Bb", "Bb", "major", "major"},
		{"C", "C", "major", "major"},
		{"C#m", "C#", "minor", "minor_natural"},
		{"Ebm", "Eb", "minor", "minor_natural"},
		{"G", "G", "major", "major"},
		{"Abm", "Ab", "minor", "minor_natural"},
		{"B", "B", "major", "major"},
		{"Bbm", "Bb", "minor", "minor_natural"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			k, err := ParseKey(tt.input)
			if err != nil {
				t.Fatalf("ParseKey(%q) error: %v", tt.input, err)
			}
			if k.Root != tt.root {
				t.Errorf("Root = %q, want %q", k.Root, tt.root)
			}
			if k.Mode != tt.mode {
				t.Errorf("Mode = %q, want %q", k.Mode, tt.mode)
			}
			if k.Scale != tt.scale {
				t.Errorf("Scale = %q, want %q", k.Scale, tt.scale)
			}
		})
	}
}

func TestParseKey_Invalid(t *testing.T) {
	invalids := []string{"", "X", "Am7", "Cmin", "abc", "123", "H#m"}
	for _, input := range invalids {
		t.Run(input, func(t *testing.T) {
			_, err := ParseKey(input)
			if err == nil {
				t.Errorf("ParseKey(%q) should return error", input)
			}
		})
	}
}

func TestParseKey_Modal(t *testing.T) {
	tests := []struct {
		input string
		root  string
		mode  string
		scale string
	}{
		{"Am-dorian", "A", "dorian", "dorian"},
		{"A-dorian", "A", "dorian", "dorian"},
		{"Em-phrygian", "E", "phrygian", "phrygian"},
		{"E-phrygian", "E", "phrygian", "phrygian"},
		{"G-mixolydian", "G", "mixolydian", "mixolydian"},
		{"C-lydian", "C", "lydian", "lydian"},
		{"F#-mixolydian", "F#", "mixolydian", "mixolydian"},
		{"Bb-dorian", "Bb", "dorian", "dorian"},
		// case-insensitive suffix
		{"Am-Dorian", "A", "dorian", "dorian"},
		{"G-MIXOLYDIAN", "G", "mixolydian", "mixolydian"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			k, err := ParseKey(tt.input)
			if err != nil {
				t.Fatalf("ParseKey(%q) unexpected error: %v", tt.input, err)
			}
			if k.Root != tt.root {
				t.Errorf("Root = %q, want %q", k.Root, tt.root)
			}
			if k.Mode != tt.mode {
				t.Errorf("Mode = %q, want %q", k.Mode, tt.mode)
			}
			if k.Scale != tt.scale {
				t.Errorf("Scale = %q, want %q", k.Scale, tt.scale)
			}
		})
	}
}

func TestParseKey_Modal_Invalid(t *testing.T) {
	invalids := []string{"A-locrian", "Am-ionian", "C-blues", "G-minor"}
	for _, input := range invalids {
		t.Run(input, func(t *testing.T) {
			_, err := ParseKey(input)
			if err == nil {
				t.Errorf("ParseKey(%q) should return error for unknown mode", input)
			}
		})
	}
}
