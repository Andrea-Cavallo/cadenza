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
	"bassline":         {minMIDI: 33, maxMIDI: 55, minActive: 8, maxActive: 13},
	"bassline_rolling": {minMIDI: 33, maxMIDI: 55, minActive: 14, maxActive: 16},
	"bassline_sub":     {minMIDI: 24, maxMIDI: 43, minActive: 4, maxActive: 6},
	"arpeggio":         {minMIDI: 48, maxMIDI: 84, minActive: 12, maxActive: 16},
	"melody":           {minMIDI: 60, maxMIDI: 96, minActive: 4, maxActive: 10},
	"chord_pad":        {minMIDI: 48, maxMIDI: 79, minActive: 2, maxActive: 6},
	"lead_stab":        {minMIDI: 60, maxMIDI: 96, minActive: 1, maxActive: 8},
}

var validStyleProfiles = map[string]bool{
	"bass_progressive": true, "bass_driving": true, "bass_sub": true, "bass_rolling": true,
	"arp_flowing": true, "arp_epic": true, "arp_staccato": true,
	"melody_expressive": true, "melody_hypnotic": true,
	"chord_pad_groove": true, "chord_pad_rolling": true, "chord_pad_sub": true,
	"lead_stab": true,
}

var validEvolutionActions = map[string]bool{
	"introduce": true, "build": true, "peak": true, "release": true,
	"octave_up": true, "octave_down": true,
	"density_up": true, "density_down": true,
	"add_chord_note": true, "strip_to_root": true, "ornament": true,
}

// Validator enforces musical rules (scale, range, density, chord coherence) on PatternSpec.
type Validator struct {
	allowedProfiles map[string]bool
}

func NewValidator() *Validator {
	return &Validator{allowedProfiles: validStyleProfiles}
}

func NewValidatorWithProfiles(profiles []string) *Validator {
	m := make(map[string]bool, len(profiles))
	for _, p := range profiles {
		m[p] = true
	}
	return &Validator{allowedProfiles: m}
}

func (v *Validator) Validate(spec *PatternSpec) error {
	return v.validate(spec, nil)
}

func (v *Validator) ValidateWithChords(spec *PatternSpec, prog theory.ChordProgression) error {
	return v.validate(spec, &prog)
}

// ValidateWithBars validates a spec with custom bar count (for --bars flag)
func (v *Validator) ValidateWithBars(spec *PatternSpec, bars int) error {
	return v.validateCustomBars(spec, nil, bars)
}

// ValidateWithChordsAndBars validates with both chord progression and custom bar count
func (v *Validator) ValidateWithChordsAndBars(spec *PatternSpec, prog theory.ChordProgression, bars int) error {
	return v.validateCustomBars(spec, &prog, bars)
}

func (v *Validator) validate(spec *PatternSpec, prog *theory.ChordProgression) error {
	return v.validateCustomBars(spec, prog, 0)
}

func (v *Validator) validateCustomBars(spec *PatternSpec, prog *theory.ChordProgression, customBars int) error {
	errs := validateMeta(spec, customBars)

	constraints, ok, typeErr := constraintsForSpec(spec)
	if !ok {
		errs = append(errs, typeErr)
		return fmt.Errorf("validation failed:\n- %s", strings.Join(errs, "\n- "))
	}

	if !v.allowedProfiles[spec.StyleProfile] {
		errs = append(errs, fmt.Sprintf("unknown style_profile %q", spec.StyleProfile))
	}

	stepErrs, activeCount := validateMotif(spec, constraints)
	errs = append(errs, stepErrs...)
	errs = append(errs, validateDensity(activeCount, constraints, spec.PatternType, len(spec.Motif.Steps))...)
	errs = append(errs, validateAntiLoop(spec)...)
	errs = append(errs, validateEvolution(spec.Evolution)...)
	errs = append(errs, v.validateOptionalChordCoherence(spec, prog)...)
	if isRollingBass(spec) {
		errs = append(errs, validateRollingChromatics(spec)...)
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed:\n- %s", strings.Join(errs, "\n- "))
	}
	return nil
}

func validateMeta(spec *PatternSpec, customBars int) []string {
	var errs []string
	if spec.Meta.BPM < 80 || spec.Meta.BPM > 150 {
		errs = append(errs, fmt.Sprintf("bpm %.1f out of range [80, 150]", spec.Meta.BPM))
	}
	if !validBarCount(spec.Meta.Bars) {
		errs = append(errs, fmt.Sprintf("bars must be 4, 16, 32, 64, or 128, got %d", spec.Meta.Bars))
	}
	if customBars > 0 && spec.Meta.Bars != customBars {
		errs = append(errs, fmt.Sprintf("bars mismatch: spec has %d, expected %d", spec.Meta.Bars, customBars))
	}
	return errs
}

func validBarCount(bars int) bool {
	return bars == 4 || bars == 16 || bars == 32 || bars == 64 || bars == 128
}

func constraintsFor(patternType string) (constraints patternConstraints, ok bool, errMsg string) {
	constraints, ok = constraintsByType[patternType]
	if !ok {
		return patternConstraints{}, false, fmt.Sprintf("unknown pattern_type %q", patternType)
	}
	return constraints, true, ""
}

// constraintsForSpec resolves constraints for a PatternSpec, overriding density bounds
// by StyleProfile for rolling and sub bass variants that share PatternType "bassline".
func constraintsForSpec(spec *PatternSpec) (constraints patternConstraints, ok bool, errMsg string) {
	if spec.PatternType == "bassline" {
		switch spec.StyleProfile {
		case "bass_rolling":
			if c, ok := constraintsByType["bassline_rolling"]; ok {
				return c, true, ""
			}
		case "bass_sub":
			if c, ok := constraintsByType["bassline_sub"]; ok {
				return c, true, ""
			}
		}
	}
	return constraintsFor(spec.PatternType)
}

// isRollingBass returns true when this spec is a rolling-style bassline.
func isRollingBass(spec *PatternSpec) bool {
	return spec.PatternType == "bassline" && spec.StyleProfile == "bass_rolling"
}

// isChromaticApproach returns true if noteName is ±1 semitone from any note in the scale.
func isChromaticApproach(noteName, key, scale string) bool {
	midi, err := theory.NoteToMIDI(noteName + "1")
	if err != nil {
		return false
	}
	pc := midi % 12
	scaleNotes, err := theory.ScaleNotes(key, scale)
	if err != nil {
		return false
	}
	for _, sn := range scaleNotes {
		snMIDI, err2 := theory.NoteToMIDI(sn + "1")
		if err2 != nil {
			continue
		}
		diff := (pc - snMIDI%12 + 12) % 12
		if diff == 1 || diff == 11 {
			return true
		}
	}
	return false
}

// validateRollingChromatics enforces: max 2 chromatic approach notes per 16-step section,
// never on the section downbeat (step 0 of each section).
func validateRollingChromatics(spec *PatternSpec) []string {
	var errs []string
	const sectionSize = 16
	totalSteps := len(spec.Motif.Steps)

	for sec := 0; sec*sectionSize < totalSteps; sec++ {
		start := sec * sectionSize
		end := start + sectionSize
		if end > totalSteps {
			end = totalSteps
		}

		chromaticCount := 0
		for i := start; i < end; i++ {
			step := spec.Motif.Steps[i]
			if !step.Active || step.Note == "" {
				continue
			}
			noteName := theory.NoteNameOnly(step.Note)
			inScale, err := theory.NoteInScale(noteName, spec.Theory.Key, spec.Theory.Scale)
			if err != nil || inScale {
				continue
			}
			if !isChromaticApproach(noteName, spec.Theory.Key, spec.Theory.Scale) {
				continue // invalid chromatic already reported by validateActiveStep
			}
			if i == start {
				errs = append(errs, fmt.Sprintf("step[%d] chromatic note %s on downbeat (not allowed for bass_rolling)", i, step.Note))
			}
			chromaticCount++
		}

		if chromaticCount > 2 {
			errs = append(errs, fmt.Sprintf("section %d: %d chromatic approach notes exceed the 2-per-section limit for bass_rolling", sec+1, chromaticCount))
		}
	}
	return errs
}

func validateMotif(spec *PatternSpec, constraints patternConstraints) (errs []string, activeCount int) {
	for i, step := range spec.Motif.Steps {
		if !step.Active {
			continue
		}
		activeCount++
		errs = append(errs, validateActiveStep(spec, constraints, i, step)...)
	}
	return errs, activeCount
}

func validateActiveStep(spec *PatternSpec, constraints patternConstraints, index int, step StepSpec) []string {
	if step.Note == "" {
		return []string{fmt.Sprintf("step[%d] active but no note", index)}
	}

	midi, err := theory.NoteToMIDI(step.Note)
	if err != nil {
		return []string{fmt.Sprintf("step[%d] invalid note %q: %v", index, step.Note, err)}
	}

	var errs []string
	if midi < constraints.minMIDI || midi > constraints.maxMIDI {
		errs = append(errs, fmt.Sprintf("step[%d] note %s (MIDI %d) out of %s range [%d, %d]",
			index, step.Note, midi, spec.PatternType, constraints.minMIDI, constraints.maxMIDI))
	}

	noteName := theory.NoteNameOnly(step.Note)
	inScale, err := theory.NoteInScale(noteName, spec.Theory.Key, spec.Theory.Scale)
	if err == nil && !inScale {
		if !isRollingBass(spec) || !isChromaticApproach(noteName, spec.Theory.Key, spec.Theory.Scale) {
			errs = append(errs, fmt.Sprintf("step[%d] note %s not in %s %s scale",
				index, step.Note, spec.Theory.Key, spec.Theory.Scale))
		}
	}
	return errs
}

func validateDensity(activeCount int, constraints patternConstraints, patternType string, motifLen int) []string {
	// Scale density limits proportionally when motif is longer than the base 16 steps.
	scale := motifLen / 16
	if scale < 1 {
		scale = 1
	}
	minActive := constraints.minActive * scale
	maxActive := constraints.maxActive * scale
	switch {
	case activeCount < minActive:
		return []string{fmt.Sprintf("density too low: %d active steps, minimum %d for %s",
			activeCount, minActive, patternType)}
	case activeCount > maxActive:
		return []string{fmt.Sprintf("density too high: %d active steps, maximum %d for %s",
			activeCount, maxActive, patternType)}
	default:
		return nil
	}
}

func validateAntiLoop(spec *PatternSpec) []string {
	repeated := repeatedPhraseCount(spec.Motif.Steps)
	if repeated == 0 {
		return nil
	}
	return []string{fmt.Sprintf("anti-loop failed: a 4-step phrase repeats more than twice without rhythmic, pitch, or articulation variation (%d excess repeat(s))", repeated)}
}

func validateEvolution(evolutions []EvolutionStep) []string {
	var errs []string
	for i, ev := range evolutions {
		if !validEvolutionActions[ev.Action] {
			errs = append(errs, fmt.Sprintf("evolution[%d] unknown action %q", i, ev.Action))
		}
		if ev.Intensity < 0 || ev.Intensity > 1 {
			errs = append(errs, fmt.Sprintf("evolution[%d] intensity %.2f out of [0, 1]", i, ev.Intensity))
		}
	}
	return errs
}

func (v *Validator) validateOptionalChordCoherence(spec *PatternSpec, prog *theory.ChordProgression) []string {
	if prog == nil || len(prog.Chords) == 0 {
		return nil
	}
	return v.validateChordCoherence(spec, *prog)
}

// Chord coherence thresholds by pattern type.
// Arpeggio is 70% (not 80%) because with 4 active steps per section, 80% would require all 4
// to be chord tones (3/4=75% < 80%), making one passing note always a hard failure.
// 70% allows 3/4 chord tones (75% ≥ 70%), which is musically correct for a real arpeggio.
var chordCoherenceThresholds = map[string]float64{
	"bassline": 0.75,
	"arpeggio": 0.70,
	"melody":   0.30,
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

		activeInSection, chordToneHits := chordToneStats(spec.Motif.Steps[start:end], chordTones)

		threshold := chordCoherenceThresholds[spec.PatternType]
		if threshold == 0 {
			threshold = 0.5
		}

		if activeInSection >= 3 && !meetsChordThreshold(chordToneHits, activeInSection, threshold) {
			ratio := float64(chordToneHits) / float64(activeInSection)
			errs = append(errs, fmt.Sprintf("section %d (bars %d-%d): chord coherence %.0f%% < required %.0f%% for %s (found %d/%d chord tones of %s%s; chord tones: %s)",
				i+1, chord.Bars[0], chord.Bars[1], ratio*100, threshold*100, spec.PatternType, chordToneHits, activeInSection, chord.Root, qualitySuffixValidator(chord.Quality), strings.Join(chordTones, ", ")))
		}
	}
	return errs
}

func chordToneStats(steps []StepSpec, chordTones []string) (active, hits int) {
	for _, step := range steps {
		if !step.Active || step.Note == "" {
			continue
		}
		active++
		if isChordTone(theory.NoteNameOnly(step.Note), chordTones) {
			hits++
		}
	}
	return active, hits
}

func isChordTone(noteName string, chordTones []string) bool {
	for _, chordTone := range chordTones {
		if strings.EqualFold(noteName, chordTone) {
			return true
		}
	}
	return false
}

func meetsChordThreshold(hits, active int, threshold float64) bool {
	return float64(hits)/float64(active) >= threshold
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
