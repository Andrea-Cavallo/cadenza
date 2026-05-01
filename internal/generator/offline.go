package generator

import (
	"crypto/sha256"
	"math"
	"strings"

	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

func offlineTemplate(patternType string, musicCtx MusicContext) *schema.PatternSpec {
	switch patternType {
	case "bassline":
		return basslineTemplate(musicCtx)
	case "arpeggio":
		return arpeggioTemplate(musicCtx)
	case "melody":
		return melodyTemplate(musicCtx)
	}
	return nil
}

const (
	offlineStyleMelodic  = "melodic"
	offlineStyleHypnotic = "hypnotic"
	offlineStyleDriving  = "driving"
	offlineStyleMinimal  = "minimal"
)

// OfflineTemplate is the public version of offlineTemplate for use by service layer.
func OfflineTemplate(patternType string, musicCtx MusicContext) *schema.PatternSpec {
	return offlineTemplate(patternType, musicCtx)
}

func basslineTemplate(ctx MusicContext) *schema.PatternSpec {
	key := ctx.Key
	prog := ctx.ChordProgression
	seed := ctx.VariationSeed
	bpm := ctx.BPM

	h := seedHash(seed + key.Root + key.Scale + "bass")
	profileName := chooseBassProfile(key, bpm, h)
	style := normalizeOfflineStyle(ctx.OfflineStyle)

	steps := make([]schema.StepSpec, 16)

	// Each 4-step section gets a DIFFERENT pattern to avoid monotony.
	// The 4th section (bars 13-16) is always simplified for release feel.
	for i, chord := range prog.Chords {
		root := chord.Root
		fifth := chordFifth(chord)
		base := i * 4

		// Use different hash byte per section so patterns vary
		pattern := h[(i*3+2)%32] % 8

		// Last section: always simplified (break section)
		if i == 3 {
			pattern = 7
		}

		color := bassColorTone(key, chord)
		switch pattern {
		case 0: // driving: root-third-fifth-root (harmonic color via third)
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: color + "2"}
			steps[base+2] = schema.StepSpec{Active: true, Note: fifth + "2"}
			steps[base+3] = schema.StepSpec{Active: true, Note: root + "2", Slide: true}
		case 1: // syncopated: root-rest-fifth-ghost
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: false}
			steps[base+2] = schema.StepSpec{Active: true, Note: fifth + "2"}
			steps[base+3] = schema.StepSpec{Active: true, Note: root + "2", Ghost: true}
		case 2: // octave bounce: root2-root3-rest-slide
			highNote := root + "3"
			if midi, _ := theory.NoteToMIDI(highNote); midi > 55 {
				highNote = root + "2"
			}
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: highNote}
			steps[base+2] = schema.StepSpec{Active: false}
			steps[base+3] = schema.StepSpec{Active: true, Note: root + "2", Slide: true}
		case 3: // long sub: root held (2 steps) + ghost
			steps[base] = schema.StepSpec{Active: true, Note: root + "1", Accent: true, Legato: true}
			steps[base+1] = schema.StepSpec{Active: false}
			steps[base+2] = schema.StepSpec{Active: true, Note: root + "2", Ghost: true}
			steps[base+3] = schema.StepSpec{Active: true, Note: fifth + "2", Slide: true}
		case 4: // harmonic walk: root-third-root-fifth (minor 3rd sounds darker)
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: color + "2"}
			steps[base+2] = schema.StepSpec{Active: true, Note: root + "2", Ghost: true}
			steps[base+3] = schema.StepSpec{Active: true, Note: fifth + "2", Slide: true}
		case 5: // off-beat pump: rest-root-rest-root
			steps[base] = schema.StepSpec{Active: false}
			steps[base+1] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+2] = schema.StepSpec{Active: false}
			steps[base+3] = schema.StepSpec{Active: true, Note: root + "2"}
		case 6: // chromatic approach: root-root-b5-slide_to_root
			approach := approachNote(root, key)
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: root + "2"}
			steps[base+2] = schema.StepSpec{Active: true, Note: approach + "2", Ghost: true}
			steps[base+3] = schema.StepSpec{Active: true, Note: root + "2", Slide: true}
		default: // break/minimal: only downbeat root
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: false}
			steps[base+2] = schema.StepSpec{Active: false}
			steps[base+3] = schema.StepSpec{Active: false}
		}
	}

	applyBassOfflineStyle(style, steps, prog, h)
	if style != offlineStyleHypnotic && style != offlineStyleMinimal {
		applyPassingNotes("bassline", steps, prog, key, h)
	}
	ensureGateVariation(steps)

	// Enforce minimum density: bassline needs at least 8 active steps.
	// If the hash selected too many sparse patterns, fill off-beats with ghost notes.
	ensureBassMinDensity(steps, prog)

	return &schema.PatternSpec{
		SpecVersion:   "1.0",
		PatternType:   "bassline",
		Meta:          schema.PatternMeta{Name: "offline-bass", BPM: bpm, Key: key.Root + modeFlag(key), Bars: 16},
		Theory:        schema.TheorySpec{Key: key.Root, Mode: key.Mode, Scale: key.Scale, OctaveRange: [2]int{1, 3}},
		StyleProfile:  profileName,
		Motif:         schema.MotifSpec{Length: 16, Steps: steps},
		Evolution:     buildBassEvolution(bpm, key),
		Automation:    schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: "medium"}},
		VariationSeed: seed,
	}
}

// ensureBassMinDensity fills inactive off-beat slots with ghost notes until
// the minimum density (8 active steps) is reached.
func ensureBassMinDensity(steps []schema.StepSpec, prog theory.ChordProgression) {
	const minActive = 8
	active := 0
	for _, s := range steps {
		if s.Active {
			active++
		}
	}
	if active >= minActive {
		return
	}
	// Fill off-beats (step 2 of each section) with ghost root notes.
	for i, chord := range prog.Chords {
		if active >= minActive {
			break
		}
		pos := i*4 + 2 // "+2" is the off-beat within the section
		if !steps[pos].Active {
			steps[pos] = schema.StepSpec{Active: true, Note: chord.Root + "2", Ghost: true}
			active++
		}
	}
	// If still short, fill remaining off-beats (step 1 of each section).
	for i, chord := range prog.Chords {
		if active >= minActive {
			break
		}
		pos := i*4 + 1
		if !steps[pos].Active {
			steps[pos] = schema.StepSpec{Active: true, Note: chord.Root + "2", Ghost: true}
			active++
		}
	}
}

// approachNote returns the diatonic scale degree immediately below the chord root.
// Using a chromatic (out-of-scale) semitone violates the validator invariant
// "notes must be in the declared scale", so we stay diatonic.
func approachNote(root string, key theory.Key) string {
	scaleNotes, err := theory.ScaleNotes(key.Root, key.Scale)
	if err != nil || len(scaleNotes) == 0 {
		return root
	}
	for i, note := range scaleNotes {
		if note == root {
			if i == 0 {
				return scaleNotes[len(scaleNotes)-1] // wrap to 7th below
			}
			return scaleNotes[i-1]
		}
	}
	return root
}

func normalizeOfflineStyle(style string) string {
	switch strings.ToLower(strings.TrimSpace(style)) {
	case offlineStyleHypnotic:
		return offlineStyleHypnotic
	case offlineStyleDriving:
		return offlineStyleDriving
	case offlineStyleMinimal:
		return offlineStyleMinimal
	default:
		return offlineStyleMelodic
	}
}

func applyBassOfflineStyle(style string, steps []schema.StepSpec, prog theory.ChordProgression, h []byte) {
	switch style {
	case offlineStyleHypnotic:
		clearSteps(steps)
		for i, chord := range prog.Chords {
			base := i * 4
			root := chord.Root + "2"
			steps[base] = schema.StepSpec{Active: true, Note: root, Accent: true, Legato: true}
			steps[base+2] = schema.StepSpec{Active: true, Note: root, Ghost: true, Legato: true}
		}
	case offlineStyleDriving:
		clearSteps(steps)
		for i, chord := range prog.Chords {
			base := i * 4
			third := chordThird(chord) + "2"
			fifth := chordFifth(chord) + "2"
			steps[base] = schema.StepSpec{Active: true, Note: chord.Root + "2", Accent: true, Staccato: h[base%len(h)]%2 == 0}
			if i == len(prog.Chords)-1 {
				continue
			}
			steps[base+1] = schema.StepSpec{Active: true, Note: third, Staccato: true}
			steps[base+2] = schema.StepSpec{Active: true, Note: fifth, Ghost: i%2 == 1}
			steps[base+3] = schema.StepSpec{Active: true, Note: chord.Root + "2", Slide: h[(base+3)%len(h)]%3 == 0}
		}
	case offlineStyleMinimal:
		clearSteps(steps)
		for i, chord := range prog.Chords {
			base := i * 4
			steps[base+1] = schema.StepSpec{Active: true, Note: chord.Root + "2", Accent: i == 0, Staccato: true}
			steps[base+3] = schema.StepSpec{Active: true, Note: chordFifth(chord) + "2", Ghost: true, Staccato: true}
		}
	default:
		mixGateArticulation(steps, h)
	}
}

func applyArpOfflineStyle(style string, steps []schema.StepSpec, prog theory.ChordProgression, h []byte) {
	switch style {
	case offlineStyleHypnotic:
		clearSteps(steps)
		for i, chord := range prog.Chords {
			base := i * 4
			notes := chordNotesOrRoot(chord)
			steps[base] = schema.StepSpec{Active: true, Note: notes[0] + "3", Accent: true, Legato: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: notes[1] + "4", Legato: true}
			steps[base+2] = schema.StepSpec{Active: true, Note: notes[2] + "4", Legato: true}
		}
	case offlineStyleDriving:
		for i, chord := range prog.Chords {
			base := i * 4
			notes := chordNotesOrRoot(chord)
			octaves := []string{"3", "4", "5", "4"}
			for j := 0; j < 4; j++ {
				note := notes[(j+int(h[(base+j)%len(h)]))%len(notes)] + octaves[j]
				steps[base+j] = schema.StepSpec{Active: true, Note: note, Accent: j == 0, Staccato: j%2 == 1}
			}
		}
	case offlineStyleMinimal:
		clearSteps(steps)
		for i, chord := range prog.Chords {
			base := i * 4
			notes := chordNotesOrRoot(chord)
			steps[base] = schema.StepSpec{Active: true, Note: notes[0] + "3", Accent: true, Staccato: true}
			steps[base+2] = schema.StepSpec{Active: true, Note: notes[2] + "4", Staccato: true}
			steps[base+3] = schema.StepSpec{Active: true, Note: notes[1] + "4", Ghost: true}
		}
	default:
		mixGateArticulation(steps, h)
	}
}

func applyMelodyOfflineStyle(style string, steps []schema.StepSpec, prog theory.ChordProgression, key theory.Key, h []byte, scaleNotes []string) {
	switch style {
	case offlineStyleHypnotic:
		clearSteps(steps)
		for i, chord := range prog.Chords {
			base := i * 4
			note := closestChordTone(chordNotesOrRoot(chord), scaleNotes, characterDegree(key)) + "4"
			if midi, _ := theory.NoteToMIDI(note); midi < 60 {
				note = theory.NoteNameOnly(note) + "5"
			}
			steps[base] = schema.StepSpec{Active: true, Note: note, Accent: i == 2, Legato: true}
		}
	case offlineStyleDriving:
		clearSteps(steps)
		for i, chord := range prog.Chords {
			base := i * 4
			notes := chordNotesOrRoot(chord)
			steps[base] = schema.StepSpec{Active: true, Note: notes[0] + "5", Accent: i == 2}
			steps[base+2] = schema.StepSpec{Active: true, Note: notes[(1+int(h[i])%2)] + "5", Staccato: true}
		}
	case offlineStyleMinimal:
		clearSteps(steps)
		for i, chord := range prog.Chords {
			base := i * 4
			steps[base+1] = schema.StepSpec{Active: true, Note: chord.Root + "4", Accent: i == 3, Staccato: true}
		}
	default:
		mixGateArticulation(steps, h)
	}
}

func applyPassingNotes(patternType string, steps []schema.StepSpec, prog theory.ChordProgression, key theory.Key, h []byte) {
	maxActive := map[string]int{"bassline": 13, "melody": 10}[patternType]
	if maxActive == 0 {
		return
	}
	chance := 15 + int(h[20])%11
	for i := range prog.Chords {
		if activeStepCount(steps) >= maxActive || int(h[(21+i)%len(h)])%100 >= chance {
			continue
		}
		base := i * 4
		for pos := base + 1; pos <= base+3 && pos < len(steps); pos++ {
			if !steps[pos].Active || steps[pos-1].Active {
				continue
			}
			target := theory.NoteNameOnly(steps[pos].Note)
			passing := passingNoteBefore(target, key)
			if passing == "" || passing == target {
				continue
			}
			steps[pos-1] = schema.StepSpec{
				Active:   true,
				Note:     passing + noteOctave(steps[pos].Note),
				Ghost:    true,
				Staccato: true,
			}
			break
		}
	}
}

func passingNoteBefore(target string, key theory.Key) string {
	midi, err := theory.NoteToMIDI(target + "4")
	if err == nil {
		candidate := theory.NoteNameOnly(theory.MIDIToNote(midi - 1))
		if inScale, scaleErr := theory.NoteInScale(candidate, key.Root, key.Scale); scaleErr == nil && inScale {
			return candidate
		}
	}
	return approachNote(target, key)
}

func noteOctave(note string) string {
	if note == "" {
		return "4"
	}
	last := note[len(note)-1]
	if last >= '0' && last <= '9' {
		return string(last)
	}
	return "4"
}

func chordNotesOrRoot(chord theory.ProgressionChord) []string {
	notes, err := theory.ChordNotes(chord.Root, chord.Quality)
	if err != nil || len(notes) < 3 {
		return []string{chord.Root, chord.Root, chord.Root}
	}
	return notes
}

func clearSteps(steps []schema.StepSpec) {
	for i := range steps {
		steps[i] = schema.StepSpec{}
	}
}

func activeStepCount(steps []schema.StepSpec) int {
	count := 0
	for _, step := range steps {
		if step.Active {
			count++
		}
	}
	return count
}

func mixGateArticulation(steps []schema.StepSpec, h []byte) {
	for i := range steps {
		if !steps[i].Active || steps[i].Slide || steps[i].Ghost {
			continue
		}
		switch h[i%len(h)] % 4 {
		case 0:
			steps[i].Legato = true
		case 1:
			steps[i].Staccato = true
		}
	}
}

func ensureGateVariation(steps []schema.StepSpec) {
	hasLegato, hasStaccato := false, false
	for _, step := range steps {
		hasLegato = hasLegato || step.Legato
		hasStaccato = hasStaccato || step.Staccato
	}
	if hasLegato && hasStaccato {
		return
	}
	for i := range steps {
		if !steps[i].Active || steps[i].Ghost || steps[i].Slide {
			continue
		}
		if !hasLegato {
			steps[i].Legato = true
			steps[i].Staccato = false
			hasLegato = true
			continue
		}
		if !hasStaccato {
			steps[i].Staccato = true
			steps[i].Legato = false
			return
		}
	}
}

func ensureMelodyMinDensity(steps []schema.StepSpec, prog theory.ChordProgression) {
	const minActive = 4
	if activeStepCount(steps) >= minActive {
		return
	}
	for i, chord := range prog.Chords {
		if activeStepCount(steps) >= minActive {
			return
		}
		pos := i * 4
		if steps[pos].Active {
			continue
		}
		notes := chordNotesOrRoot(chord)
		steps[pos] = schema.StepSpec{Active: true, Note: notes[0] + "5", Accent: i == 2, Legato: true}
	}
}

func arpeggioTemplate(ctx MusicContext) *schema.PatternSpec {
	key := ctx.Key
	prog := ctx.ChordProgression
	seed := ctx.VariationSeed
	bpm := ctx.BPM

	h := seedHash(seed + key.Root + key.Scale + "arp")
	profileName := chooseArpProfile(key, bpm, h)
	style := normalizeOfflineStyle(ctx.OfflineStyle)

	steps := make([]schema.StepSpec, 16)

	// Open voicing technique (Prydz/Opus style):
	// Root in low octave, 3rd/5th spread across higher octaves
	voicingStyle := h[5] % 4
	prevPattern := byte(255)

	for i, chord := range prog.Chords {
		notes, err := theory.ChordNotes(chord.Root, chord.Quality)
		if err != nil || len(notes) < 3 {
			notes = []string{chord.Root, chord.Root, chord.Root}
		}

		// Build open voicing pool: root low, tensions high
		var pool []string
		switch voicingStyle {
		case 0: // wide open: root3, 5th4, 3rd5, root5
			pool = []string{notes[0] + "3", notes[2] + "4", notes[1] + "5", notes[0] + "5"}
		case 1: // 1st inversion spread: 3rd3, 5th4, root5, 3rd5
			pool = []string{notes[1] + "3", notes[2] + "4", notes[0] + "5", notes[1] + "5"}
		case 2: // 2nd inversion spread: 5th3, root4, 3rd5, 5th5
			pool = []string{notes[2] + "3", notes[0] + "4", notes[1] + "5", notes[2] + "5"}
		default: // drop-2: root3, 3rd4, 5th4, root5
			pool = []string{notes[0] + "3", notes[1] + "4", notes[2] + "4", notes[0] + "5"}
		}

		base := i * 4

		arpPattern := chooseArpPattern(h, key, i)
		if i > 0 && arpPattern == prevPattern {
			arpPattern = (arpPattern + 1) % 6
		}
		prevPattern = arpPattern

		switch arpPattern {
		case 0: // ascending legato (classic Prydz)
			steps[base] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: pool[1], Legato: true}
			steps[base+2] = schema.StepSpec{Active: true, Note: pool[2], Legato: true}
			steps[base+3] = schema.StepSpec{Active: true, Note: pool[3], Legato: true}
		case 1: // descending with accent top
			steps[base] = schema.StepSpec{Active: true, Note: pool[3], Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: pool[2], Legato: true}
			steps[base+2] = schema.StepSpec{Active: true, Note: pool[1], Legato: true}
			steps[base+3] = schema.StepSpec{Active: true, Note: pool[0], Legato: true}
		case 2: // pendulum wide
			steps[base] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: pool[3], Legato: true}
			steps[base+2] = schema.StepSpec{Active: true, Note: pool[1], Legato: true}
			steps[base+3] = schema.StepSpec{Active: true, Note: pool[2], Legato: true}
		case 3: // octave pulse (root low-high alternating)
			steps[base] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: pool[3]}
			steps[base+2] = schema.StepSpec{Active: true, Note: pool[0]}
			steps[base+3] = schema.StepSpec{Active: true, Note: pool[3], Legato: true}
		case 4: // broken with rest (tension)
			steps[base] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: pool[2], Legato: true}
			steps[base+2] = schema.StepSpec{Active: false}
			steps[base+3] = schema.StepSpec{Active: true, Note: pool[3], Legato: true}
		default: // climbing 5ths
			steps[base] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: pool[1], Legato: true}
			steps[base+2] = schema.StepSpec{Active: true, Note: pool[3], Legato: true}
			steps[base+3] = schema.StepSpec{Active: true, Note: pool[2], Legato: true}
		}
	}

	applyArpOfflineStyle(style, steps, prog, h)
	ensureGateVariation(steps)

	sweepStyle := "medium"
	if bpm >= 124 || key.Mode == "minor" {
		sweepStyle = "dramatic"
	}

	return &schema.PatternSpec{
		SpecVersion:   "1.0",
		PatternType:   "arpeggio",
		Meta:          schema.PatternMeta{Name: "offline-arp", BPM: bpm, Key: key.Root + modeFlag(key), Bars: 16},
		Theory:        schema.TheorySpec{Key: key.Root, Mode: key.Mode, Scale: key.Scale, OctaveRange: [2]int{3, 6}},
		StyleProfile:  profileName,
		Motif:         schema.MotifSpec{Length: 16, Steps: steps},
		Evolution:     buildArpEvolution(bpm, key),
		Automation:    schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: sweepStyle}},
		VariationSeed: seed,
	}
}

func melodyTemplate(ctx MusicContext) *schema.PatternSpec {
	key := ctx.Key
	prog := ctx.ChordProgression
	seed := ctx.VariationSeed
	bpm := ctx.BPM

	h := seedHash(seed + key.Root + key.Scale + "melody")
	profileName := chooseMelodyProfile(key, bpm, h)
	style := normalizeOfflineStyle(ctx.OfflineStyle)

	scaleNotes, _ := theory.ScaleNotes(key.Root, key.Scale)
	if len(scaleNotes) < 7 {
		scaleNotes = []string{"C", "D", "E", "F", "G", "A", "B"}
	}

	steps := make([]schema.StepSpec, 16)

	// Hypnotic approach: short motif (3-5 notes) repeated with micro-variation.
	// Use intervals of 4ths and 5ths for open electronic sound.
	motifLen := 3 + int(h[6])%3 // 3, 4, or 5 notes

	// Build a motif from chord tones + scale tensions using wide intervals
	motifDegrees := buildHypnoticMotif(h, motifLen, scaleNotes)

	// Rhythm patterns: true = note onset, false = rest/sustain
	// Hypnotic techno uses few notes with space — notes are long (legato gate handles duration)
	rhythms := [][]bool{
		{true, false, false, false, true, false, false, false, true, false, false, false, true, false, false, false},  // 4 long notes (quarter feel)
		{true, false, false, true, false, false, true, false, false, true, false, false, false, false, false, false},  // dotted 8th feel
		{true, false, false, false, false, true, false, false, false, false, true, false, false, false, false, false}, // very sparse (3 notes)
		{true, false, false, true, false, false, false, true, false, false, true, false, false, false, true, false},   // syncopated
		{true, false, false, false, true, false, true, false, false, false, false, false, true, false, false, false},  // call-response
		{true, false, true, false, false, false, true, false, false, false, false, true, false, false, false, false},  // triplet-ish push
		{true, false, false, false, false, false, true, false, false, false, true, false, false, true, false, false},  // tension-hold
	}
	rhythmIdx := int(h[10]) % len(rhythms)
	rhythm := rhythms[rhythmIdx]

	motifNoteIdx := 0
	for i := 0; i < 16; i++ {
		if !rhythm[i] {
			steps[i] = schema.StepSpec{Active: false}
			continue
		}

		// Pick note from motif cycle
		chordIdx := i / 4
		chord := prog.Chords[chordIdx]
		chordTones, _ := theory.ChordNotes(chord.Root, chord.Quality)

		degree := normalizeDegree(shapeMelodyDegree(motifDegrees[motifNoteIdx%len(motifDegrees)], motifNoteIdx, key, h), len(scaleNotes))
		noteName := scaleNotes[degree]

		// Every other note gravitates toward chord tones
		if motifNoteIdx%2 == 0 && len(chordTones) > 0 {
			noteName = closestChordTone(chordTones, scaleNotes, degree)
		}

		// Octave: keep in the 4-5 range for melodic techno feel
		octave := "4"
		if degree >= 4 {
			octave = "5"
		}
		midiVal, _ := theory.NoteToMIDI(noteName + octave)
		if midiVal < 60 {
			octave = "5"
		} else if midiVal > 84 {
			octave = "4"
		}

		isAccent := i%4 == 0
		// Long notes: legato for sustained feel
		isLegato := motifNoteIdx > 0

		steps[i] = schema.StepSpec{
			Active: true,
			Note:   noteName + octave,
			Accent: isAccent,
			Legato: isLegato,
		}
		motifNoteIdx++
	}

	applyMelodyOfflineStyle(style, steps, prog, key, h, scaleNotes)
	if style != offlineStyleHypnotic && style != offlineStyleMinimal {
		applyPassingNotes("melody", steps, prog, key, h)
	}
	ensureMelodyMinDensity(steps, prog)
	ensureGateVariation(steps)

	return &schema.PatternSpec{
		SpecVersion:   "1.0",
		PatternType:   "melody",
		Meta:          schema.PatternMeta{Name: "offline-melody", BPM: bpm, Key: key.Root + modeFlag(key), Bars: 16},
		Theory:        schema.TheorySpec{Key: key.Root, Mode: key.Mode, Scale: key.Scale, OctaveRange: [2]int{4, 6}},
		StyleProfile:  profileName,
		Motif:         schema.MotifSpec{Length: 16, Steps: steps},
		Evolution:     buildMelodyEvolution(bpm, key),
		Automation:    schema.AutomationIntent{ModWheel: &schema.ModWheelIntent{Style: "subtle"}},
		VariationSeed: seed,
	}
}

// buildHypnoticMotif creates a short motif using 4ths/5ths intervals for open sound.
func buildHypnoticMotif(h []byte, length int, scaleNotes []string) []int {
	numDegrees := len(scaleNotes)
	motif := make([]int, length)

	// Start on a strong scale degree (root, 4th, or 5th)
	startDegrees := []int{0, 3, 4}
	motif[0] = startDegrees[int(h[14])%len(startDegrees)]

	// Build using wide intervals (3rds, 4ths, 5ths)
	wideIntervals := []int{3, 4, 5, -3, -4, -2}
	for i := 1; i < length; i++ {
		interval := wideIntervals[int(h[15+i])%len(wideIntervals)]
		next := (motif[i-1] + interval) % numDegrees
		if next < 0 {
			next += numDegrees
		}
		motif[i] = next
	}

	return motif
}

func chooseArpPattern(h []byte, key theory.Key, section int) byte {
	characterBias := byte(characterDegree(key) + len(key.Root))
	return (h[(section+4)%32] + characterBias + byte(section)) % 6
}

func shapeMelodyDegree(degree, motifNoteIdx int, key theory.Key, h []byte) int {
	contour := int(h[9]+byte(characterDegree(key))) % 3
	switch contour {
	case 0: // ascending arc
		return degree + motifNoteIdx/2
	case 1: // descending resolution
		return degree - motifNoteIdx/2
	default: // tension-hold around the mode's character note
		if motifNoteIdx%3 == 1 {
			return characterDegree(key)
		}
		return degree
	}
}

func bassColorTone(key theory.Key, chord theory.ProgressionChord) string {
	chordTones, err := theory.ChordNotes(chord.Root, chord.Quality)
	if err != nil || len(chordTones) == 0 {
		return chordThird(chord)
	}

	color := scaleDegreeName(key, characterDegree(key))
	for _, tone := range chordTones {
		if tone == color {
			return color
		}
	}
	return chordThird(chord)
}

func scaleDegreeName(key theory.Key, degree int) string {
	scaleNotes, err := theory.ScaleNotes(key.Root, key.Scale)
	if err != nil || len(scaleNotes) == 0 {
		return key.Root
	}
	return scaleNotes[normalizeDegree(degree, len(scaleNotes))]
}

func normalizeDegree(degree, length int) int {
	if length <= 0 {
		return 0
	}
	degree %= length
	if degree < 0 {
		degree += length
	}
	return degree
}

func characterDegree(key theory.Key) int {
	switch key.Scale {
	case "minor_natural", "minor_harmonic":
		return 5 // minor 6th, the darker Aeolian color
	case "dorian":
		return 5 // raised 6th
	case "phrygian":
		return 1 // flat 2nd
	case "mixolydian":
		return 6 // flat 7th
	case "lydian":
		return 3 // raised 4th
	default:
		return 4 // stable fifth for plain major
	}
}

// buildBassEvolution creates evolution for bass — full energy, minimal density changes.
func buildBassEvolution(bpm float64, key theory.Key) []schema.EvolutionStep {
	if bpm >= 128 {
		return []schema.EvolutionStep{
			{FromBar: 1, ToBar: 4, Action: "build", Intensity: 0.7},
			{FromBar: 5, ToBar: 8, Action: "build", Intensity: 0.8},
			{FromBar: 9, ToBar: 12, Action: "peak", Intensity: 1.0},
			{FromBar: 13, ToBar: 16, Action: "release", Intensity: 0.6},
		}
	}
	return []schema.EvolutionStep{
		{FromBar: 1, ToBar: 4, Action: "introduce", Intensity: 0.5},
		{FromBar: 5, ToBar: 8, Action: "build", Intensity: 0.7},
		{FromBar: 9, ToBar: 12, Action: "peak", Intensity: 0.9},
		{FromBar: 13, ToBar: 16, Action: "release", Intensity: 0.5},
	}
}

// buildArpEvolution creates evolution for arpeggio — density crescendo typical of Prydz.
func buildArpEvolution(bpm float64, key theory.Key) []schema.EvolutionStep {
	return []schema.EvolutionStep{
		{FromBar: 1, ToBar: 4, Action: "introduce", Intensity: 0.3},
		{FromBar: 5, ToBar: 8, Action: "density_up", Intensity: 0.5},
		{FromBar: 9, ToBar: 12, Action: "peak", Intensity: 0.9},
		{FromBar: 13, ToBar: 16, Action: "build", Intensity: 0.8},
	}
}

// buildMelodyEvolution creates evolution for melody — sparse intro, phrase peak, fade.
func buildMelodyEvolution(bpm float64, key theory.Key) []schema.EvolutionStep {
	return []schema.EvolutionStep{
		{FromBar: 1, ToBar: 4, Action: "introduce", Intensity: 0.4},
		{FromBar: 5, ToBar: 8, Action: "build", Intensity: 0.6},
		{FromBar: 9, ToBar: 12, Action: "peak", Intensity: 0.8},
		{FromBar: 13, ToBar: 16, Action: "release", Intensity: 0.4},
	}
}

func chooseBassProfile(key theory.Key, bpm float64, h []byte) string {
	// BPM-based selection with Key influence
	if bpm < 118 {
		// Deep/ambient territory — prefer sub-bass or progressive
		candidates := []string{"bass_sub", "bass_progressive"}
		return candidates[int(h[0])%len(candidates)]
	}
	if bpm >= 128 {
		// High-energy techno — prefer driving
		candidates := []string{"bass_driving", "bass_progressive"}
		if key.Mode == "minor" {
			return "bass_driving"
		}
		return candidates[int(h[0])%len(candidates)]
	}
	// Mid-range 118-127 — progressive house territory
	candidates := []string{"bass_progressive"}
	if key.Mode == "minor" {
		candidates = append(candidates, "bass_driving")
	}
	return candidates[int(h[0])%len(candidates)]
}

func chooseArpProfile(key theory.Key, bpm float64, h []byte) string {
	// BPM-based selection with Key influence
	if bpm < 118 {
		// Deep/ambient — prefer flowing
		return "arp_flowing"
	}
	if bpm > 128 {
		// High-energy — prefer staccato or epic
		candidates := []string{"arp_epic", "arp_staccato"}
		return candidates[int(h[1])%len(candidates)]
	}
	// Progressive house 118-128 — epic for minor, flowing for major
	if key.Mode == "minor" {
		candidates := []string{"arp_epic"}
		if bpm >= 124 {
			candidates = append(candidates, "arp_staccato")
		}
		return candidates[int(h[1])%len(candidates)]
	}
	// Major keys — mix epic and flowing
	candidates := []string{"arp_flowing", "arp_epic"}
	return candidates[int(h[1])%len(candidates)]
}

func chooseMelodyProfile(key theory.Key, bpm float64, h []byte) string {
	// BPM-based selection with Key influence
	if bpm < 118 {
		// Deep/ambient — expressive with space
		return "melody_expressive"
	}
	// Techno/progressive 118+ — hypnotic for minor, mixed for major
	if key.Mode == "minor" {
		// Minor keys strongly prefer hypnotic
		return "melody_hypnotic"
	}
	// Major keys — mix based on BPM
	if bpm >= 124 {
		candidates := []string{"melody_hypnotic", "melody_expressive"}
		return candidates[int(h[2])%len(candidates)]
	}
	return "melody_expressive"
}

func chordFifth(chord theory.ProgressionChord) string {
	notes, err := theory.ChordNotes(chord.Root, chord.Quality)
	if err != nil || len(notes) < 3 {
		return chord.Root
	}
	return notes[2]
}

func chordThird(chord theory.ProgressionChord) string {
	notes, err := theory.ChordNotes(chord.Root, chord.Quality)
	if err != nil || len(notes) < 2 {
		return chord.Root
	}
	return notes[1]
}

func closestChordTone(chordTones, scaleNotes []string, degree int) string {
	targetNote := scaleNotes[degree%len(scaleNotes)]
	targetMIDI, err := theory.NoteToMIDI(targetNote + "4")
	if err != nil {
		return chordTones[0]
	}

	best := chordTones[0]
	bestDist := math.MaxInt32
	for _, ct := range chordTones {
		ctMIDI, err := theory.NoteToMIDI(ct + "4")
		if err != nil {
			continue
		}
		dist := int(math.Abs(float64(ctMIDI - targetMIDI)))
		if dist > 6 {
			dist = 12 - dist
		}
		if dist < bestDist {
			bestDist = dist
			best = ct
		}
	}
	return best
}

func seedHash(seed string) []byte {
	h := sha256.Sum256([]byte(seed))
	return h[:]
}
