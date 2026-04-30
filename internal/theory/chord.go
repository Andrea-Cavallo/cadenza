package theory

import "fmt"

// Chord represents a musical chord with root, quality, and note names.
type Chord struct {
	Root    string
	Quality string   // "major", "minor", "dim", "aug", "sus2", "sus4"
	Notes   []string // note names without octave
}

var chordIntervals = map[string][]int{
	"major": {0, 4, 7},
	"minor": {0, 3, 7},
	"dim":   {0, 3, 6},
	"aug":   {0, 4, 8},
	"sus2":  {0, 2, 7},
	"sus4":  {0, 5, 7},
}

var scaleChordQualities = map[string][]string{
	"major":          {"major", "minor", "minor", "major", "major", "minor", "dim"},
	"minor_natural":  {"minor", "dim", "major", "minor", "minor", "major", "major"},
	"minor_harmonic": {"minor", "dim", "aug", "minor", "major", "major", "dim"},
}

func ChordNotes(root, quality string) ([]string, error) {
	intervals, ok := chordIntervals[quality]
	if !ok {
		return nil, fmt.Errorf("unknown chord quality %q", quality)
	}

	rootSem, err := rootSemitone(root)
	if err != nil {
		return nil, err
	}

	chromatic := chromaticSharps
	if useFlats[root] {
		chromatic = chromaticFlats
	}

	notes := make([]string, len(intervals))
	for i, interval := range intervals {
		idx := (rootSem + interval) % 12
		notes[i] = chromatic[idx]
	}
	return notes, nil
}

func chordsInKey(root, scaleType string) ([]Chord, error) {
	qualities, ok := scaleChordQualities[scaleType]
	if !ok {
		return nil, fmt.Errorf("no chord qualities for scale %q", scaleType)
	}

	scaleNotes, err := ScaleNotes(root, scaleType)
	if err != nil {
		return nil, err
	}

	chords := make([]Chord, len(scaleNotes))
	for i, noteName := range scaleNotes {
		notes, err := ChordNotes(noteName, qualities[i])
		if err != nil {
			return nil, fmt.Errorf("chord on degree %d: %w", i+1, err)
		}
		chords[i] = Chord{Root: noteName, Quality: qualities[i], Notes: notes}
	}
	return chords, nil
}
