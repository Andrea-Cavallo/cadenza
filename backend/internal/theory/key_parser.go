package theory

import (
	"fmt"
	"strings"
)

// Key represents a parsed musical key with root note, mode, and scale type.
type Key struct {
	Root  string // "A", "D", "F#", "Bb"
	Mode  string // "major", "minor", "dorian", "phrygian", "mixolydian", "lydian"
	Scale string // matches a key in scaleIntervals: "major", "minor_natural", "dorian", ...
}

var validRoots = map[string]bool{
	"C": true, "C#": true, "Db": true,
	"D": true, "D#": true, "Eb": true,
	"E": true,
	"F": true, "F#": true, "Gb": true,
	"G": true, "G#": true, "Ab": true,
	"A": true, "A#": true, "Bb": true,
	"B": true,
}

// validModes maps the suffix after the root (and optional "m") to (mode, scale) pairs.
// Supported input forms:
//
//	"Am"           → minor (Aeolian)
//	"A"            → major (Ionian)
//	"Am-dorian"    → Dorian (minor-flavored, raised 6th)
//	"A-dorian"     → same
//	"Em-phrygian"  → Phrygian (flat 2nd)
//	"E-phrygian"   → same
//	"G-mixolydian" → Mixolydian (flat 7th)
//	"C-lydian"     → Lydian (raised 4th)
var modalSuffixes = map[string][2]string{
	"-dorian":     {"dorian", "dorian"},
	"m-dorian":    {"dorian", "dorian"},
	"-phrygian":   {"phrygian", "phrygian"},
	"m-phrygian":  {"phrygian", "phrygian"},
	"-mixolydian": {"mixolydian", "mixolydian"},
	"-lydian":     {"lydian", "lydian"},
}

func ParseKey(input string) (Key, error) {
	if len(input) == 0 {
		return Key{}, fmt.Errorf("empty key string")
	}

	// Normalise: trim spaces, lowercase the suffix for mode names.
	// Root note capitalisation is preserved.
	input = strings.TrimSpace(input)

	var root string
	var rest string

	if len(input) >= 2 && (input[1] == '#' || input[1] == 'b') {
		root = input[:2]
		rest = input[2:]
	} else {
		root = input[:1]
		rest = input[1:]
	}

	if !validRoots[root] {
		return Key{}, fmt.Errorf("invalid root note %q in key %q", root, input)
	}

	// Lowercase the suffix so "Am-Dorian" and "Am-dorian" both work.
	restLower := strings.ToLower(rest)

	switch restLower {
	case "":
		return Key{Root: root, Mode: "major", Scale: "major"}, nil
	case "m":
		return Key{Root: root, Mode: "minor", Scale: "minor_natural"}, nil
	}

	if pair, ok := modalSuffixes[restLower]; ok {
		return Key{Root: root, Mode: pair[0], Scale: pair[1]}, nil
	}

	return Key{}, fmt.Errorf(
		"invalid key suffix %q in %q — accepted suffixes: (none)=major, m=minor, "+
			"-dorian, -phrygian, -mixolydian, -lydian (with optional leading 'm')",
		rest, input,
	)
}
