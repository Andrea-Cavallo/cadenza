package theory

import (
	"fmt"
	"strconv"
)

var noteNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

var noteToSemitone = map[string]int{
	"C": 0, "C#": 1, "Db": 1,
	"D": 2, "D#": 3, "Eb": 3,
	"E": 4, "Fb": 4,
	"F": 5, "F#": 6, "Gb": 6,
	"G": 7, "G#": 8, "Ab": 8,
	"A": 9, "A#": 10, "Bb": 10,
	"B": 11, "Cb": 11,
}

func NoteToMIDI(note string) (int, error) {
	if len(note) < 2 {
		return 0, fmt.Errorf("invalid note %q: too short", note)
	}

	var name string
	var octaveStr string

	if len(note) >= 3 && (note[1] == '#' || note[1] == 'b') {
		name = note[:2]
		octaveStr = note[2:]
	} else {
		name = note[:1]
		octaveStr = note[1:]
	}

	semitone, ok := noteToSemitone[name]
	if !ok {
		return 0, fmt.Errorf("invalid note name %q in %q", name, note)
	}

	octave, err := strconv.Atoi(octaveStr)
	if err != nil || octave < 0 || octave > 9 {
		return 0, fmt.Errorf("invalid octave %q in %q", octaveStr, note)
	}

	midi := (octave+1)*12 + semitone
	if midi < 0 || midi > 127 {
		return 0, fmt.Errorf("MIDI number %d out of range for %q", midi, note)
	}
	return midi, nil
}

func MIDIToNote(midi int) string {
	if midi < 0 || midi > 127 {
		return fmt.Sprintf("MIDI(%d)", midi)
	}
	octave := (midi / 12) - 1
	semitone := midi % 12
	return fmt.Sprintf("%s%d", noteNames[semitone], octave)
}

// NoteNameOnly strips the octave from a note string (e.g. "A#3" → "A#").
func NoteNameOnly(note string) string {
	if len(note) >= 3 && (note[1] == '#' || note[1] == 'b') {
		return note[:2]
	}
	if len(note) >= 1 {
		return note[:1]
	}
	return note
}
