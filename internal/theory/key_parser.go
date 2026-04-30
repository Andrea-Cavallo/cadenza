package theory

import "fmt"

// Key represents a parsed musical key with root note, mode, and scale type.
type Key struct {
	Root  string // "A", "D", "F#", "Bb"
	Mode  string // "major", "minor"
	Scale string // "major", "minor_natural"
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

func ParseKey(input string) (Key, error) {
	if len(input) == 0 {
		return Key{}, fmt.Errorf("empty key string")
	}

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

	switch rest {
	case "":
		return Key{Root: root, Mode: "major", Scale: "major"}, nil
	case "m":
		return Key{Root: root, Mode: "minor", Scale: "minor_natural"}, nil
	default:
		return Key{}, fmt.Errorf("invalid key suffix %q in %q (expected nothing or 'm')", rest, input)
	}
}
