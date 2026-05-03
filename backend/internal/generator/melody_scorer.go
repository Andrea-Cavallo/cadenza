package generator

import (
	"math"

	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// MelodyPhraseScore measures musical quality of an offline melody.
// Used only in tests and diagnostics — not a production gate.
type MelodyPhraseScore struct {
	RestRatio         float64
	PickupPresence    float64
	ContourScore      float64
	ChordToneStrength float64
	RegisterSep       float64
	PhraseScore       float64
}

// ScoreMelodyPhrase computes the MelodyPhraseScore for a melody spec.
// arpSteps is used for RegisterSep; may be nil (RegisterSep = 1.0).
func ScoreMelodyPhrase(melodySpec *schema.PatternSpec, arpSteps []schema.StepSpec, prog theory.ChordProgression, key theory.Key) MelodyPhraseScore {
	steps := melodySpec.Motif.Steps
	total := len(steps)
	if total == 0 {
		return MelodyPhraseScore{}
	}

	// RestRatio: globally across all 64 steps.
	inactive := 0
	for _, s := range steps {
		if !s.Active {
			inactive++
		}
	}
	restRatio := float64(inactive) / float64(total)

	// PickupPresence: sections with >= 1 ghost+staccato step.
	numSections := total / stepsPerSection
	sectionsWithPickup := 0
	for sec := 0; sec < numSections; sec++ {
		base := sec * stepsPerSection
		for i := base; i < base+stepsPerSection; i++ {
			if steps[i].Active && steps[i].Ghost && steps[i].Staccato {
				sectionsWithPickup++
				break
			}
		}
	}
	pickupPresence := float64(sectionsWithPickup) / float64(numSections)

	// ContourScore: sections without double same-direction leap > 5 semitones.
	sectionsOK := 0
	for sec := 0; sec < numSections; sec++ {
		base := sec * stepsPerSection
		if !hasDoubleLeap(steps, base, base+stepsPerSection) {
			sectionsOK++
		}
	}
	contourScore := float64(sectionsOK) / float64(numSections)

	// ChordToneStrength.
	scaleNotes, _ := theory.ScaleNotes(key.Root, key.Scale)
	charDeg := characterDegree(key)
	var charNote string
	if len(scaleNotes) > 0 {
		charNote = scaleNotes[normalizeDegree(charDeg, len(scaleNotes))]
	}

	chordToneHits, activeNonGhost := 0, 0
	for i, chord := range prog.Chords {
		base := i * stepsPerSection
		ct, _ := theory.ChordNotes(chord.Root, chord.Quality)
		ctSet := make(map[string]bool, len(ct)+1)
		for _, n := range ct {
			ctSet[n] = true
		}
		if charNote != "" {
			ctSet[charNote] = true
		}
		for j := base; j < base+stepsPerSection && j < total; j++ {
			if !steps[j].Active || steps[j].Ghost {
				continue
			}
			activeNonGhost++
			noteName := steps[j].Note
			if len(noteName) > 0 {
				last := noteName[len(noteName)-1]
				if last >= '0' && last <= '9' {
					noteName = noteName[:len(noteName)-1]
				}
			}
			if ctSet[noteName] {
				chordToneHits++
			}
		}
	}
	chordToneStrength := 1.0
	if activeNonGhost > 0 {
		chordToneStrength = float64(chordToneHits) / float64(activeNonGhost)
	}

	// RegisterSep: melody steps not colliding with arp in octave 5 range (MIDI 72-83).
	melodyActive, collisions := 0, 0
	for i, ms := range steps {
		if !ms.Active {
			continue
		}
		melodyActive++
		mMIDI, err := theory.NoteToMIDI(ms.Note)
		if err != nil || mMIDI < 72 || mMIDI > 83 {
			continue
		}
		if arpSteps != nil && i < len(arpSteps) && arpSteps[i].Active {
			aMIDI, err := theory.NoteToMIDI(arpSteps[i].Note)
			if err == nil && aMIDI >= 72 && aMIDI <= 83 {
				collisions++
			}
		}
	}
	registerSep := 1.0
	if melodyActive > 0 {
		registerSep = 1.0 - float64(collisions)/float64(melodyActive)
	}

	restScore := scoreRestRatio(restRatio)
	phraseScore := restScore*0.20 +
		pickupPresence*0.20 +
		contourScore*0.25 +
		chordToneStrength*0.25 +
		registerSep*0.10

	return MelodyPhraseScore{
		RestRatio:         restRatio,
		PickupPresence:    pickupPresence,
		ContourScore:      contourScore,
		ChordToneStrength: chordToneStrength,
		RegisterSep:       registerSep,
		PhraseScore:       phraseScore,
	}
}

func scoreRestRatio(r float64) float64 {
	if r >= 0.50 && r <= 0.70 {
		return 1.0
	}
	if r < 0.50 {
		return r / 0.50
	}
	return math.Max(0, 1.0-(r-0.70)/0.30)
}

// hasDoubleLeap returns true if there are 3 consecutive non-ghost active notes
// with same-direction leaps both > 5 semitones.
func hasDoubleLeap(steps []schema.StepSpec, from, to int) bool {
	var midiSeq []int
	for i := from; i < to && i < len(steps); i++ {
		if !steps[i].Active || steps[i].Ghost {
			continue
		}
		m, err := theory.NoteToMIDI(steps[i].Note)
		if err == nil {
			midiSeq = append(midiSeq, m)
		}
	}
	for i := 0; i < len(midiSeq)-2; i++ {
		d1 := midiSeq[i+1] - midiSeq[i]
		d2 := midiSeq[i+2] - midiSeq[i+1]
		a1, a2 := d1, d2
		if a1 < 0 {
			a1 = -a1
		}
		if a2 < 0 {
			a2 = -a2
		}
		sameDir := intSign(d1) == intSign(d2) && intSign(d1) != 0
		if sameDir && a1 > 5 && a2 > 5 {
			return true
		}
	}
	return false
}

func intSign(x int) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}
