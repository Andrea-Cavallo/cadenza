package generator

import (
	"crypto/sha256"
	"encoding/binary"

	midipkg "github.com/Andrea-Cavallo/cadenza/internal/midi"
)

const (
	// General MIDI drum channel (1-indexed in MIDI spec, 0-indexed in code)
	drumChannel = 9

	// General MIDI drum note numbers
	kickNote     = 36 // C1
	snareNote    = 38 // D1
	clapNote     = 39 // D#1
	closedHHNote = 42 // F#1
	openHHNote   = 46 // A#1
)

// GenerateDrumPattern creates a techno/progressive house drum pattern
func GenerateDrumPattern(bpm float64, bars int, seed string) []midipkg.MIDIEvent {
	resolution := 120 // ticks per step (16th note)
	stepsPerBar := 16
	totalSteps := bars * stepsPerBar

	events := make([]midipkg.MIDIEvent, 0, totalSteps*4)

	// Use seed to deterministically vary the pattern
	h := sha256.Sum256([]byte(seed + "drums"))
	rng := int(binary.BigEndian.Uint32(h[:4]))

	for step := 0; step < totalSteps; step++ {
		tick := int64(step * resolution)
		barPos := step % stepsPerBar

		// Kick: 4-on-the-floor with occasional variation
		if barPos%4 == 0 {
			velocity := uint8(100)
			if barPos == 0 {
				velocity = 110 // accent downbeat
			}
			events = append(events, createDrumHit(tick, kickNote, velocity))
		} else if bars > 16 && (step+rng)%32 == 16 {
			// Add syncopated kicks in longer arrangements
			events = append(events, createDrumHit(tick, kickNote, 85))
		}

		// Clap/Snare: on 2 and 4
		if barPos == 4 || barPos == 12 {
			vel := uint8(95)
			if (step/stepsPerBar)%4 == 3 {
				// Every 4th bar, accent the snare
				vel = 105
			}
			events = append(events, createDrumHit(tick, clapNote, vel))
		}

		// Closed hi-hat: steady 16ths with velocity variation
		if barPos%2 == 0 {
			vel := uint8(60)
			if barPos%4 == 0 {
				vel = 70 // accent on beats
			}
			events = append(events, createDrumHit(tick, closedHHNote, vel))
		}

		// Open hi-hat: occasional off-beat emphasis
		if bars > 16 && barPos == 6 && (step/stepsPerBar)%8 == 7 {
			events = append(events, createDrumHit(tick, openHHNote, 75))
		}
	}

	return events
}

func createDrumHit(tick int64, note, velocity uint8) midipkg.MIDIEvent {
	return midipkg.MIDIEvent{
		Type:     midipkg.NoteOn,
		Tick:     tick,
		Channel:  drumChannel,
		Note:     note,
		Velocity: velocity,
	}
}

// Note: NoteOff events for drums are typically not needed as most drum sounds
// are one-shots. If required, they can be added at tick + duration.
