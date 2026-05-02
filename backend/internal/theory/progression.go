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
	degrees []int // scale degree indices (0-based, I=0 … VII=6)
}

// ─── Natural minor (Aeolian) ────────────────────────────────────────────────
// Degree map: 0=i(min) 1=ii°(dim) 2=III(maj) 3=iv(min) 4=v(min) 5=VI(maj) 6=VII(maj)
var minorProgressions = []progressionTemplate{
	{degrees: []int{0, 5, 2, 6}}, // i – VI – III – VII  (Am-F-C-G classic)
	{degrees: []int{0, 3, 5, 4}}, // i – iv – VI – v
	{degrees: []int{0, 6, 5, 4}}, // i – VII – VI – v    (descending aeolian)
	{degrees: []int{0, 2, 6, 5}}, // i – III – VII – VI
	{degrees: []int{0, 3, 6, 2}}, // i – iv – VII – III  (dark driving)
	{degrees: []int{0, 5, 6, 0}}, // i – VI – VII – i    (circular / hypnotic)
	{degrees: []int{0, 2, 5, 6}}, // i – III – VI – VII  (minor with lift)
	{degrees: []int{0, 3, 0, 6}}, // i – iv – i – VII    (minimal loop)
	{degrees: []int{0, 5, 3, 4}}, // i – VI – iv – v     (romantic minor)
	{degrees: []int{0, 6, 3, 5}}, // i – VII – iv – VI   (progressive minor)
	{degrees: []int{0, 4, 5, 6}}, // i – v – VI – VII    (natural minor descent)
	{degrees: []int{0, 5, 2, 3}}, // i – VI – III – iv   (aeolian color)
}

// ─── Major (Ionian) ─────────────────────────────────────────────────────────
// Degree map: 0=I(maj) 1=ii(min) 2=iii(min) 3=IV(maj) 4=V(maj) 5=vi(min) 6=vii°(dim)
var majorProgressions = []progressionTemplate{
	{degrees: []int{0, 4, 5, 3}}, // I – V – vi – IV   (axis)
	{degrees: []int{0, 5, 3, 4}}, // I – vi – IV – V
	{degrees: []int{0, 3, 5, 4}}, // I – IV – vi – V
	{degrees: []int{0, 2, 3, 4}}, // I – iii – IV – V
	{degrees: []int{0, 4, 3, 5}}, // I – V – IV – vi   (pop cadence variant)
	{degrees: []int{0, 3, 0, 4}}, // I – IV – I – V    (strong tonic)
	{degrees: []int{0, 5, 4, 3}}, // I – vi – V – IV   (descending retro)
	{degrees: []int{0, 2, 5, 4}}, // I – iii – vi – V
	{degrees: []int{0, 3, 2, 5}}, // I – IV – iii – vi  (descending thirds)
	{degrees: []int{0, 4, 2, 5}}, // I – V – iii – vi
	{degrees: []int{0, 5, 2, 3}}, // I – vi – iii – IV  (melancholic major)
	{degrees: []int{0, 1, 3, 4}}, // I – ii – IV – V   (basic cadential)
}

// ─── Dorian ─────────────────────────────────────────────────────────────────
// Degree map: 0=i(min) 1=ii(min) 2=bIII(maj) 3=IV(maj) 4=v(min) 5=vi°(dim) 6=bVII(maj)
// Character: IV is MAJOR and bVII is MAJOR — the two defining chords.
var dorianProgressions = []progressionTemplate{
	{degrees: []int{0, 3, 0, 3}}, // i – IV – i – IV    (the Dorian vamp)
	{degrees: []int{0, 3, 6, 0}}, // i – IV – bVII – i
	{degrees: []int{0, 6, 3, 0}}, // i – bVII – IV – i
	{degrees: []int{0, 1, 3, 6}}, // i – ii – IV – bVII
	{degrees: []int{0, 3, 2, 6}}, // i – IV – bIII – bVII
	{degrees: []int{0, 1, 6, 3}}, // i – ii – bVII – IV
	{degrees: []int{0, 3, 1, 6}}, // i – IV – ii – bVII
	{degrees: []int{0, 6, 3, 1}}, // i – bVII – IV – ii
}

// ─── Phrygian ───────────────────────────────────────────────────────────────
// Degree map: 0=i(min) 1=bII(maj) 2=bIII(maj) 3=iv(min) 4=v°(dim) 5=bVI(maj) 6=bVII(min)
// Character: bII is MAJOR — the Andalusian tension chord.
var phrygianProgressions = []progressionTemplate{
	{degrees: []int{0, 1, 0, 1}}, // i – bII – i – bII  (Andalusian cadence)
	{degrees: []int{0, 1, 2, 1}}, // i – bII – bIII – bII
	{degrees: []int{0, 6, 5, 1}}, // i – bVII – bVI – bII (descending Phrygian)
	{degrees: []int{1, 0, 6, 0}}, // bII – i – bVII – i
	{degrees: []int{0, 5, 1, 0}}, // i – bVI – bII – i
	{degrees: []int{0, 3, 1, 0}}, // i – iv – bII – i
	{degrees: []int{0, 1, 5, 2}}, // i – bII – bVI – bIII
	{degrees: []int{0, 6, 1, 0}}, // i – bVII – bII – i
}

// ─── Mixolydian ─────────────────────────────────────────────────────────────
// Degree map: 0=I(maj) 1=ii(min) 2=iii°(dim) 3=IV(maj) 4=v(min) 5=vi(min) 6=bVII(maj)
// Character: bVII is MAJOR — the festival/rock defining chord.
var mixolydianProgressions = []progressionTemplate{
	{degrees: []int{0, 6, 3, 0}}, // I – bVII – IV – I  (classic Mixolydian)
	{degrees: []int{0, 3, 6, 0}}, // I – IV – bVII – I
	{degrees: []int{0, 6, 0, 3}}, // I – bVII – I – IV  (festival energy)
	{degrees: []int{0, 1, 3, 6}}, // I – ii – IV – bVII
	{degrees: []int{0, 3, 1, 6}}, // I – IV – ii – bVII
	{degrees: []int{0, 5, 6, 3}}, // I – vi – bVII – IV
	{degrees: []int{0, 6, 5, 3}}, // I – bVII – vi – IV
	{degrees: []int{0, 4, 3, 6}}, // I – v – IV – bVII
}

// ─── Lydian ─────────────────────────────────────────────────────────────────
// Degree map: 0=I(maj) 1=II(maj) 2=iii(min) 3=#iv°(dim) 4=V(maj) 5=vi(min) 6=vii(min)
// Character: II is MAJOR — the floating, ethereal sound.
var lydianProgressions = []progressionTemplate{
	{degrees: []int{0, 1, 0, 4}}, // I – II – I – V    (floating Lydian)
	{degrees: []int{0, 4, 1, 0}}, // I – V – II – I
	{degrees: []int{0, 1, 4, 0}}, // I – II – V – I
	{degrees: []int{0, 2, 1, 4}}, // I – iii – II – V
	{degrees: []int{0, 1, 5, 4}}, // I – II – vi – V
	{degrees: []int{0, 4, 5, 1}}, // I – V – vi – II
	{degrees: []int{0, 1, 2, 5}}, // I – II – iii – vi
	{degrees: []int{0, 5, 1, 4}}, // I – vi – II – V
}

func progressionPool(root, scaleType string) []ChordProgression {
	diatonic, err := chordsInKey(root, scaleType)
	if err != nil {
		return nil
	}

	var templates []progressionTemplate
	mode := scaleType
	switch scaleType {
	case "major":
		templates = majorProgressions
	case "minor_natural", "minor":
		templates = minorProgressions
		mode = "minor"
	case "dorian":
		templates = dorianProgressions
	case "phrygian":
		templates = phrygianProgressions
	case "mixolydian":
		templates = mixolydianProgressions
	case "lydian":
		templates = lydianProgressions
	default:
		templates = minorProgressions
		mode = "minor"
	}

	barRanges := [][2]int{{1, 4}, {5, 8}, {9, 12}, {13, 16}}
	pool := make([]ChordProgression, len(templates))

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
