package theory

import "fmt"

var scaleIntervals = map[string][]int{
	"major":          {0, 2, 4, 5, 7, 9, 11},
	"minor_natural":  {0, 2, 3, 5, 7, 8, 10},
	"minor_harmonic": {0, 2, 3, 5, 7, 8, 11},
	"dorian":         {0, 2, 3, 5, 7, 9, 10},
	"phrygian":       {0, 1, 3, 5, 7, 8, 10},
}

var chromaticSharps = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
var chromaticFlats = []string{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"}

var useFlats = map[string]bool{
	"F": true, "Bb": true, "Eb": true, "Ab": true, "Db": true, "Gb": true,
}

func rootSemitone(root string) (int, error) {
	sem, ok := noteToSemitone[root]
	if !ok {
		return 0, fmt.Errorf("unknown root %q", root)
	}
	return sem, nil
}

func ScaleNotes(root, scaleType string) ([]string, error) {
	intervals, ok := scaleIntervals[scaleType]
	if !ok {
		return nil, fmt.Errorf("unknown scale type %q", scaleType)
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

func NoteInScale(note, root, scaleType string) (bool, error) {
	scaleNotes, err := ScaleNotes(root, scaleType)
	if err != nil {
		return false, err
	}

	noteSem, ok := noteToSemitone[note]
	if !ok {
		return false, fmt.Errorf("unknown note %q", note)
	}

	for _, sn := range scaleNotes {
		snSem := noteToSemitone[sn]
		if noteSem == snSem {
			return true, nil
		}
	}
	return false, nil
}
