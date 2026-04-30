package theory

import (
	"crypto/sha256"
	"encoding/binary"
)

// ChordProgression is the harmonic contract shared by all pattern generators.
type ChordProgression struct {
	Chords []ProgressionChord `json:"chords"`
	Key    string             `json:"key"`
	Mode   string             `json:"mode"`
}

// ProgressionChord represents a single chord within a progression, spanning a bar range.
type ProgressionChord struct {
	Root    string   `json:"root"`
	Quality string   `json:"quality"`
	Notes   []string `json:"notes"`
	Bars    [2]int   `json:"bars"` // [1,4], [5,8], [9,12], [13,16]
}

type progressionTemplate struct {
	degrees []int // scale degree indices (0-based)
}

var minorProgressions = []progressionTemplate{
	{degrees: []int{0, 5, 2, 6}}, // i → VI → III → VII
	{degrees: []int{0, 3, 5, 4}}, // i → iv → VI → V
	{degrees: []int{0, 6, 5, 4}}, // i → VII → VI → v
	{degrees: []int{0, 2, 6, 5}}, // i → III → VII → VI
}

var majorProgressions = []progressionTemplate{
	{degrees: []int{0, 4, 5, 3}}, // I → V → vi → IV
	{degrees: []int{0, 5, 3, 4}}, // I → vi → IV → V
	{degrees: []int{0, 3, 5, 4}}, // I → IV → vi → V
	{degrees: []int{0, 2, 3, 4}}, // I → iii → IV → V
}

func progressionPool(root, scaleType string) []ChordProgression {
	diatonic, err := chordsInKey(root, scaleType)
	if err != nil {
		return nil
	}

	var templates []progressionTemplate
	mode := "minor"
	if scaleType == "major" {
		templates = majorProgressions
		mode = "major"
	} else {
		templates = minorProgressions
	}

	pool := make([]ChordProgression, len(templates))
	barRanges := [][2]int{{1, 4}, {5, 8}, {9, 12}, {13, 16}}

	for i, tmpl := range templates {
		chords := make([]ProgressionChord, 4)
		for j, deg := range tmpl.degrees {
			c := diatonic[deg]
			chords[j] = ProgressionChord{
				Root:    c.Root,
				Quality: c.Quality,
				Notes:   c.Notes,
				Bars:    barRanges[j],
			}
		}
		pool[i] = ChordProgression{Chords: chords, Key: root, Mode: mode}
	}
	return pool
}

func SelectProgression(root, scaleType, seed string) ChordProgression {
	pool := progressionPool(root, scaleType)
	if len(pool) == 0 {
		return ChordProgression{Key: root, Mode: "minor"}
	}
	h := sha256.Sum256([]byte(seed + root + scaleType))
	idx := int(binary.BigEndian.Uint32(h[:4])) % len(pool)
	return pool[idx]
}
