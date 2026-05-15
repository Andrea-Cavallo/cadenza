package generator

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"github.com/Andrea-Cavallo/cadenza/internal/llm"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// repairConstraints mirrors the validator's constraintsByType table.
// minMIDI, maxMIDI, minActive, maxActive
var repairConstraints = map[string][4]int{
	"bassline": {33, 55, 8, 13},
	"arpeggio": {48, 84, 12, 16},
	"melody":   {60, 96, 4, 10},
}

// repairChordThresholds mirrors the validator's chordCoherenceThresholds.
var repairChordThresholds = map[string]float64{
	"bassline": 0.75,
	"arpeggio": 0.70,
	"melody":   0.30,
}

var repairDefaultStyleProfiles = map[string]string{
	"bassline": "bass_progressive",
	"arpeggio": "arp_flowing",
	"melody":   "melody_expressive",
}

var repairValidStyleProfiles = map[string]bool{
	"bass_progressive": true, "bass_driving": true, "bass_sub": true,
	"arp_flowing": true, "arp_epic": true, "arp_staccato": true,
	"melody_expressive": true, "melody_hypnotic": true,
}

var repairValidEvolutionActions = map[string]bool{
	"introduce": true, "build": true, "peak": true, "release": true,
	"octave_up": true, "octave_down": true,
	"density_up": true, "density_down": true,
	"add_chord_note": true, "strip_to_root": true, "ornament": true,
}

// repairSpec applies mechanical fixes to a PatternSpec that failed LLM validation.
// Handles: invalid style_profile, evolution action/intensity errors, out-of-range notes,
// density violations, and chord coherence failures.
// Returns the marshalled fixed spec only if the result passes validate.
func repairSpec(raw []byte, musicCtx MusicContext, validate llm.ValidateFunc) ([]byte, error) {
	var spec schema.PatternSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, fmt.Errorf("repair: unmarshal: %w", err)
	}

	spec.Theory.Key = musicCtx.Key.Root
	spec.Theory.Scale = musicCtx.Key.Scale

	c, ok := repairConstraints[spec.PatternType]
	if !ok {
		return nil, fmt.Errorf("repair: unknown pattern type %q", spec.PatternType)
	}
	if len(spec.Motif.Steps) == 0 {
		return nil, fmt.Errorf("repair: empty motif")
	}

	slog.Debug("repair: starting",
		"type", spec.PatternType,
		"style_profile", spec.StyleProfile,
		"bpm", spec.Meta.BPM,
		"bars", spec.Meta.Bars,
		"steps", len(spec.Motif.Steps),
		"evolution_steps", len(spec.Evolution),
		"active_before", countActiveSteps(spec.Motif.Steps),
	)

	// Fix structural fields that the validator checks before musical rules.
	repairMetaFields(&spec, musicCtx)
	repairStyleProfile(&spec)
	repairEvolution(&spec)

	// Fix musical content.
	scaleNotes, _ := theory.ScaleNotes(musicCtx.Key.Root, musicCtx.Key.Scale)
	repairNoteRanges(&spec, c)
	repairDensity(&spec, c, scaleNotes)
	if len(musicCtx.ChordProgression.Chords) > 0 {
		repairChordCoherence(&spec, musicCtx.ChordProgression, c)
	}

	slog.Debug("repair: finished",
		"type", spec.PatternType,
		"style_profile", spec.StyleProfile,
		"active_after", countActiveSteps(spec.Motif.Steps),
	)

	repaired, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("repair: marshal: %w", err)
	}
	if err := validate(repaired); err != nil {
		slog.Debug("repair: post-repair validation failed", "type", spec.PatternType, "error", err)
		return nil, fmt.Errorf("repair: still invalid after repair: %w", err)
	}
	return repaired, nil
}

// repairMetaFields ensures BPM, Bars, and SpecVersion match expected values.
func repairMetaFields(spec *schema.PatternSpec, musicCtx MusicContext) {
	if spec.SpecVersion == "" {
		slog.Debug("repair: set spec_version", "type", spec.PatternType, "value", "1.0")
		spec.SpecVersion = "1.0"
	}
	if spec.Meta.BPM < 80 || spec.Meta.BPM > 150 {
		slog.Debug("repair: bpm clamped", "type", spec.PatternType, "from", spec.Meta.BPM, "to", musicCtx.BPM)
		spec.Meta.BPM = musicCtx.BPM
	}
	validBars := map[int]bool{4: true, 16: true, 32: true, 64: true, 128: true}
	if !validBars[spec.Meta.Bars] {
		slog.Debug("repair: bars fixed", "type", spec.PatternType, "from", spec.Meta.Bars, "to", musicCtx.Bars)
		spec.Meta.Bars = musicCtx.Bars
	}
}

// repairStyleProfile replaces an invalid style_profile with the correct default for the pattern type.
// Small LLMs often output the pattern type name (e.g. "bassline") instead of a valid profile.
func repairStyleProfile(spec *schema.PatternSpec) {
	if repairValidStyleProfiles[spec.StyleProfile] {
		return
	}
	if d, ok := repairDefaultStyleProfiles[spec.PatternType]; ok {
		slog.Debug("repair: style_profile fixed", "type", spec.PatternType, "from", spec.StyleProfile, "to", d)
		spec.StyleProfile = d
	}
}

// repairEvolution fixes invalid action strings and out-of-range intensities.
// If the evolution array is empty it inserts a default 4-phase arc.
func repairEvolution(spec *schema.PatternSpec) {
	if len(spec.Evolution) == 0 {
		slog.Debug("repair: evolution empty, inserting default arc", "type", spec.PatternType)
		spec.Evolution = []schema.EvolutionStep{
			{FromBar: 1, ToBar: 4, Action: "introduce", Intensity: 0.3},
			{FromBar: 5, ToBar: 8, Action: "build", Intensity: 0.6},
			{FromBar: 9, ToBar: 12, Action: "peak", Intensity: 0.9},
			{FromBar: 13, ToBar: 16, Action: "release", Intensity: 0.5},
		}
		return
	}
	for i, ev := range spec.Evolution {
		if !repairValidEvolutionActions[ev.Action] {
			fixed := defaultEvolutionAction(i, len(spec.Evolution))
			slog.Debug("repair: evolution action fixed", "type", spec.PatternType, "index", i, "from", ev.Action, "to", fixed)
			spec.Evolution[i].Action = fixed
		}
		// LLMs sometimes output 0–100 scale instead of 0.0–1.0
		if ev.Intensity > 1 {
			scaled := ev.Intensity / 100
			slog.Debug("repair: evolution intensity scaled", "type", spec.PatternType, "index", i, "from", ev.Intensity, "to", scaled)
			spec.Evolution[i].Intensity = scaled
		}
		if spec.Evolution[i].Intensity < 0 {
			slog.Debug("repair: evolution intensity clamped to 0", "type", spec.PatternType, "index", i, "was", spec.Evolution[i].Intensity)
			spec.Evolution[i].Intensity = 0
		}
		if spec.Evolution[i].Intensity > 1 {
			slog.Debug("repair: evolution intensity clamped to 1", "type", spec.PatternType, "index", i, "was", spec.Evolution[i].Intensity)
			spec.Evolution[i].Intensity = 1
		}
	}
}

func defaultEvolutionAction(idx, total int) string {
	if total == 0 {
		return "introduce"
	}
	pos := float64(idx) / float64(total)
	switch {
	case pos < 0.25:
		return "introduce"
	case pos < 0.5:
		return "build"
	case pos < 0.75:
		return "peak"
	default:
		return "release"
	}
}

// repairNoteRanges shifts notes out of the valid MIDI range by octave until they land inside it.
func repairNoteRanges(spec *schema.PatternSpec, c [4]int) {
	for i, step := range spec.Motif.Steps {
		if !step.Active || step.Note == "" {
			continue
		}
		midi, err := theory.NoteToMIDI(step.Note)
		if err != nil {
			slog.Debug("repair: invalid note deactivated", "type", spec.PatternType, "step", i, "note", step.Note, "error", err)
			spec.Motif.Steps[i].Active = false
			continue
		}
		originalMIDI := midi
		for midi < c[0] {
			midi += 12
		}
		for midi > c[1] {
			midi -= 12
		}
		if midi < c[0] || midi > c[1] {
			slog.Debug("repair: note out of range, deactivated", "type", spec.PatternType, "step", i, "note", step.Note, "midi", originalMIDI, "range", fmt.Sprintf("[%d,%d]", c[0], c[1]))
			spec.Motif.Steps[i].Active = false
		} else if midi != originalMIDI {
			fixed := theory.MIDIToNote(midi)
			slog.Debug("repair: note shifted by octave", "type", spec.PatternType, "step", i, "from", step.Note, "to", fixed, "from_midi", originalMIDI, "to_midi", midi)
			spec.Motif.Steps[i].Note = fixed
		}
	}
}

// repairDensity activates or deactivates steps to bring active count into [minActive, maxActive].
// Activations are distributed one-per-section first to also help the anti-loop check.
func repairDensity(spec *schema.PatternSpec, c [4]int, scaleNotes []string) {
	active := countActiveSteps(spec.Motif.Steps)
	total := len(spec.Motif.Steps)
	slog.Debug("repair: density check", "type", spec.PatternType, "active", active, "min", c[2], "max", c[3], "total_steps", total)

	if active < c[2] && len(scaleNotes) > 0 {
		center := (c[0] + c[1]) / 2
		rootNote := closestNoteInRange(scaleNotes[0], c[0], c[1], center)
		if rootNote != "" {
			slog.Debug("repair: density too low, activating steps", "type", spec.PatternType, "active", active, "need", c[2], "fill_note", rootNote)
			sectionSize := total / 4
			if sectionSize < 1 {
				sectionSize = 1
			}
			// First pass: activate one step per section (spread activations out)
			for s := 0; s < 4 && active < c[2]; s++ {
				start := s * sectionSize
				end := start + sectionSize
				if end > total {
					end = total
				}
				for i := start; i < end; i++ {
					if !spec.Motif.Steps[i].Active {
						spec.Motif.Steps[i].Active = true
						spec.Motif.Steps[i].Note = rootNote
						slog.Debug("repair: step activated (spread)", "type", spec.PatternType, "step", i, "section", s, "note", rootNote)
						active++
						break
					}
				}
			}
			// Second pass: fill remaining if still not enough
			for i := range spec.Motif.Steps {
				if active >= c[2] {
					break
				}
				if !spec.Motif.Steps[i].Active {
					spec.Motif.Steps[i].Active = true
					spec.Motif.Steps[i].Note = rootNote
					slog.Debug("repair: step activated (fill)", "type", spec.PatternType, "step", i, "note", rootNote)
					active++
				}
			}
		}
	}

	// Deactivate excess steps (keep accented ones last)
	if active > c[3] {
		slog.Debug("repair: density too high, deactivating steps", "type", spec.PatternType, "active", active, "max", c[3])
		for i := len(spec.Motif.Steps) - 1; i >= 0 && active > c[3]; i-- {
			if spec.Motif.Steps[i].Active && !spec.Motif.Steps[i].Accent {
				slog.Debug("repair: step deactivated", "type", spec.PatternType, "step", i, "note", spec.Motif.Steps[i].Note)
				spec.Motif.Steps[i].Active = false
				active--
			}
		}
	}
	slog.Debug("repair: density after fix", "type", spec.PatternType, "active", active)
}

// repairChordCoherence replaces non-chord-tone notes in each section with the nearest chord tone
// until the section meets the coherence threshold.
func repairChordCoherence(spec *schema.PatternSpec, prog theory.ChordProgression, c [4]int) {
	threshold := repairChordThresholds[spec.PatternType]
	if threshold == 0 {
		threshold = 0.5
	}
	total := len(spec.Motif.Steps)
	stepsPerSection := total / len(prog.Chords)
	if stepsPerSection == 0 {
		return
	}

	for si, chord := range prog.Chords {
		chordTones, err := theory.ChordNotes(chord.Root, chord.Quality)
		if err != nil || len(chordTones) == 0 {
			continue
		}
		start := si * stepsPerSection
		end := start + stepsPerSection
		if end > total {
			end = total
		}

		active, hits := sectionChordToneStats(spec.Motif.Steps[start:end], chordTones)
		if active < 3 {
			continue
		}
		needed := int(math.Ceil(float64(active)*threshold)) - hits
		coherencePct := 0.0
		if active > 0 {
			coherencePct = float64(hits) / float64(active) * 100
		}
		slog.Debug("repair: chord coherence check",
			"type", spec.PatternType,
			"section", si+1,
			"chord", fmt.Sprintf("%s%s", chord.Root, chord.Quality),
			"chord_tones", strings.Join(chordTones, ","),
			"active", active,
			"hits", hits,
			"coherence_pct", fmt.Sprintf("%.0f%%", coherencePct),
			"needed", needed,
		)
		if needed <= 0 {
			continue
		}

		for j := start; j < end && needed > 0; j++ {
			step := spec.Motif.Steps[j]
			if !step.Active || step.Note == "" {
				continue
			}
			if isChordToneName(theory.NoteNameOnly(step.Note), chordTones) {
				continue
			}
			midi, err := theory.NoteToMIDI(step.Note)
			if err != nil {
				continue
			}
			newNote := nearestChordToneNote(midi, chordTones, c)
			if newNote == "" {
				continue
			}
			slog.Debug("repair: chord tone replacement",
				"type", spec.PatternType,
				"section", si+1,
				"step", j,
				"from", step.Note,
				"to", newNote,
				"chord", fmt.Sprintf("%s%s", chord.Root, chord.Quality),
			)
			spec.Motif.Steps[j].Note = newNote
			needed--
		}
	}
}

func sectionChordToneStats(steps []schema.StepSpec, chordTones []string) (active, hits int) {
	for _, s := range steps {
		if !s.Active || s.Note == "" {
			continue
		}
		active++
		if isChordToneName(theory.NoteNameOnly(s.Note), chordTones) {
			hits++
		}
	}
	return
}

func isChordToneName(noteName string, chordTones []string) bool {
	for _, ct := range chordTones {
		if strings.EqualFold(noteName, ct) {
			return true
		}
	}
	return false
}

// nearestChordToneNote finds the chord tone closest to midi within [c[0], c[1]] and returns the
// note as "Name+octave" using the chord tone's own spelling (flat or sharp), so that post-repair
// string comparison against the chord tone list always matches.
// Example: chord tone "Ab" at MIDI 44 → "Ab2" (not "G#2" which MIDIToNote would give).
func nearestChordToneNote(midi int, chordTones []string, c [4]int) string {
	best := -1
	bestDist := 1000
	bestName := ""
	for _, ct := range chordTones {
		sem := noteNameSemitone(ct)
		for oct := 0; oct <= 8; oct++ {
			m := (oct+1)*12 + sem
			if m < c[0] || m > c[1] {
				continue
			}
			d := repairAbs(m - midi)
			if d < bestDist {
				bestDist = d
				best = m
				bestName = fmt.Sprintf("%s%d", ct, (m/12)-1)
			}
		}
	}
	if best < 0 {
		return ""
	}
	return bestName
}

func countActiveSteps(steps []schema.StepSpec) int {
	n := 0
	for _, s := range steps {
		if s.Active {
			n++
		}
	}
	return n
}

// closestNoteInRange finds the octave of noteName (no octave) closest to center within [minMIDI, maxMIDI].
// Returns the note using the original noteName spelling to avoid enharmonic mismatches
// (e.g. "Ab" stays "Ab2", not "G#2" which theory.MIDIToNote would produce).
func closestNoteInRange(noteName string, minMIDI, maxMIDI, center int) string {
	sem := noteNameSemitone(noteName)
	best := -1
	bestDist := 1000
	for oct := 0; oct <= 8; oct++ {
		m := (oct+1)*12 + sem
		if m < minMIDI || m > maxMIDI {
			continue
		}
		d := repairAbs(m - center)
		if d < bestDist {
			bestDist = d
			best = m
		}
	}
	if best < 0 {
		return ""
	}
	return fmt.Sprintf("%s%d", noteName, (best/12)-1)
}

func noteNameSemitone(name string) int {
	midi, err := theory.NoteToMIDI(name + "4")
	if err != nil {
		return 0
	}
	return midi % 12
}

func repairAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
