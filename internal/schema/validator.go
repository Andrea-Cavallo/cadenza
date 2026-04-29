package schema

import (
	"fmt"
	"strings"

	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

type patternConstraints struct {
	minMIDI   int
	maxMIDI   int
	minActive int
	maxActive int
}

var constraintsByType = map[string]patternConstraints{
	"bassline": {minMIDI: 33, maxMIDI: 55, minActive: 8, maxActive: 13},
	"arpeggio": {minMIDI: 48, maxMIDI: 84, minActive: 12, maxActive: 16},
	"melody":   {minMIDI: 60, maxMIDI: 96, minActive: 4, maxActive: 10},
}

var validStyleProfiles = map[string]bool{
	"bass_progressive": true, "bass_techno_driving": true,
	"bass_melodic_dark": true, "bass_deep": true,
	"arp_flowing": true, "arp_rhythmic": true,
	"arp_ambient": true, "arp_plucky": true,
	"melody_expressive": true, "melody_minimal": true,
	"melody_soaring": true, "melody_rhythmic": true,
}

var validEvolutionActions = map[string]bool{
	"introduce": true, "build": true, "peak": true, "release": true,
	"octave_up": true, "octave_down": true,
	"density_up": true, "density_down": true,
	"add_chord_note": true, "strip_to_root": true, "ornament": true,
}

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) Validate(spec *PatternSpec) error {
	return v.validate(spec, nil)
}

func (v *Validator) ValidateWithChords(spec *PatternSpec, prog theory.ChordProgression) error {
	return v.validate(spec, &prog)
}

func (v *Validator) validate(spec *PatternSpec, prog *theory.ChordProgression) error {
	var errs []string

	if spec.Meta.BPM < 80 || spec.Meta.BPM > 150 {
		errs = append(errs, fmt.Sprintf("bpm %.1f out of range [80, 150]", spec.Meta.BPM))
	}

	if spec.Meta.Bars != 16 {
		errs = append(errs, fmt.Sprintf("bars must be 16, got %d", spec.Meta.Bars))
	}

	constraints, ok := constraintsByType[spec.PatternType]
	if !ok {
		errs = append(errs, fmt.Sprintf("unknown pattern_type %q", spec.PatternType))
		return fmt.Errorf("validation failed:\n- %s", strings.Join(errs, "\n- "))
	}

	if !validStyleProfiles[spec.StyleProfile] {
		errs = append(errs, fmt.Sprintf("unknown style_profile %q", spec.StyleProfile))
	}

	activeCount := 0
	for i, step := range spec.Motif.Steps {
		if !step.Active {
			continue
		}
		activeCount++

		if step.Note == "" {
			errs = append(errs, fmt.Sprintf("step[%d] active but no note", i))
			continue
		}

		midi, err := theory.NoteToMIDI(step.Note)
		if err != nil {
			errs = append(errs, fmt.Sprintf("step[%d] invalid note %q: %v", i, step.Note, err))
			continue
		}

		if midi < constraints.minMIDI || midi > constraints.maxMIDI {
			errs = append(errs, fmt.Sprintf("step[%d] note %s (MIDI %d) out of %s range [%d, %d]",
				i, step.Note, midi, spec.PatternType, constraints.minMIDI, constraints.maxMIDI))
		}

		noteName := theory.NoteNameOnly(step.Note)
		inScale, err := theory.NoteInScale(noteName, spec.Theory.Key, spec.Theory.Scale)
		if err == nil && !inScale {
			errs = append(errs, fmt.Sprintf("step[%d] note %s not in %s %s scale",
				i, step.Note, spec.Theory.Key, spec.Theory.Scale))
		}
	}

	if activeCount < constraints.minActive {
		errs = append(errs, fmt.Sprintf("density too low: %d active steps, minimum %d for %s",
			activeCount, constraints.minActive, spec.PatternType))
	}
	if activeCount > constraints.maxActive {
		errs = append(errs, fmt.Sprintf("density too high: %d active steps, maximum %d for %s",
			activeCount, constraints.maxActive, spec.PatternType))
	}

	for i, ev := range spec.Evolution {
		if !validEvolutionActions[ev.Action] {
			errs = append(errs, fmt.Sprintf("evolution[%d] unknown action %q", i, ev.Action))
		}
		if ev.Intensity < 0 || ev.Intensity > 1 {
			errs = append(errs, fmt.Sprintf("evolution[%d] intensity %.2f out of [0, 1]", i, ev.Intensity))
		}
	}

	if prog != nil && len(prog.Chords) > 0 {
		errs = append(errs, v.validateChordCoherence(spec, *prog)...)
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed:\n- %s", strings.Join(errs, "\n- "))
	}
	return nil
}

func (v *Validator) validateChordCoherence(spec *PatternSpec, prog theory.ChordProgression) []string {
	var errs []string
	stepsPerSection := len(spec.Motif.Steps) / len(prog.Chords)

	for i, chord := range prog.Chords {
		chordTones, err := theory.ChordNotes(chord.Root, chord.Quality)
		if err != nil || len(chordTones) == 0 {
			continue
		}

		start := i * stepsPerSection
		end := start + stepsPerSection
		if end > len(spec.Motif.Steps) {
			end = len(spec.Motif.Steps)
		}

		activeInSection := 0
		chordToneHits := 0
		for j := start; j < end; j++ {
			step := spec.Motif.Steps[j]
			if !step.Active || step.Note == "" {
				continue
			}
			activeInSection++
			noteName := theory.NoteNameOnly(step.Note)
			for _, ct := range chordTones {
				if strings.EqualFold(noteName, ct) {
					chordToneHits++
					break
				}
			}
		}

		if activeInSection >= 3 && chordToneHits == 0 {
			errs = append(errs, fmt.Sprintf("section %d (bars %d-%d): no chord tones of %s%s found in %d active notes",
				i+1, chord.Bars[0], chord.Bars[1], chord.Root, qualitySuffixValidator(chord.Quality), activeInSection))
		}
	}
	return errs
}

func qualitySuffixValidator(q string) string {
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

