package theory

import (
	"fmt"
	"strings"
)

func ParseChordName(name string) (root, quality string, err error) {
	if len(name) == 0 {
		return "", "", fmt.Errorf("empty chord name")
	}

	root = string(name[0])
	rest := ""

	if len(name) > 1 {
		if name[1] == '#' || name[1] == 'b' {
			root = name[:2]
			if len(name) > 2 {
				rest = name[2:]
			}
		} else {
			rest = name[1:]
		}
	}

	if _, err := NoteToMIDI(root + "4"); err != nil {
		return "", "", fmt.Errorf("invalid root note %q", root)
	}

	rest = strings.ToLower(rest)
	switch rest {
	case "", "maj":
		quality = "major"
	case "m", "min":
		quality = "minor"
	case "dim":
		quality = "dim"
	case "aug":
		quality = "aug"
	case "sus2":
		quality = "sus2"
	case "sus4":
		quality = "sus4"
	default:
		return "", "", fmt.Errorf("unknown chord quality %q", rest)
	}

	return root, quality, nil
}

func QualitySuffix(q string) string {
	switch q {
	case "minor":
		return "m"
	case "dim":
		return "dim"
	case "aug":
		return "aug"
	default:
		return ""
	}
}
