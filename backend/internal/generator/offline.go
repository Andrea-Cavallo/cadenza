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

	// motifSteps is the number of steps in the expanded motif (4 chords × 16 steps each).
	// 64 steps = 4 bars at 480 ticks/beat, 120 ticks/step.
	// Meta.Bars = 4 renderer iterations → 4 × 4 bars = 16 musical bars total.
	motifSteps = 64

	// stepsPerSection is the number of steps allocated per chord section in the motif.
	stepsPerSection = motifSteps / 4 // = 16
)

// ContourType descrive la forma melodica di una sezione di 16 step.
type ContourType string

const (
	ContourArch           ContourType = "arch"
	ContourQuestionAnswer ContourType = "question_answer"
	ContourTensionHold    ContourType = "tension_hold"
	ContourDescRelease    ContourType = "descending_release"
)

// stepRole è la funzione musicale di uno step attivo.
type stepRole string

const (
	roleTarget stepRole = "target"
	rolePickup stepRole = "pickup"
	roleEcho   stepRole = "echo"
	roleFill   stepRole = "fill"
)

// StepIntention è la decisione del contour pass per una posizione.
type StepIntention struct {
	Active     bool
	Role       stepRole
	Direction  int  // -1 down, 0 neutral, +1 up
	PreferHigh bool // preferisce ottava 6 su 5
}

// SectionContour tiene le 16 intenzioni per una sezione accordo.
type SectionContour struct {
	Type         ContourType
	TargetDegree int
	Intentions   [stepsPerSection]StepIntention
}

// MelodyContour raccoglie i contour delle 4 sezioni.
type MelodyContour struct {
	Sections [4]SectionContour
}

// Template ritmici: 0=rest, 1=target, 2=pickup, 3=echo, 4=fill.
// 3 varianti per ContourType, scelte via seed hash.
var archRhythms = [3][stepsPerSection]byte{
	{1, 0, 1, 0, 1, 0, 0, 1, 1, 0, 1, 0, 0, 3, 0, 2}, // V0: 8 attivi
	{1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 3, 0, 0}, // V1: 7 attivi
	{1, 2, 0, 0, 1, 0, 1, 0, 1, 0, 0, 1, 0, 0, 3, 0}, // V2: 7 attivi
}

var questionAnswerRhythms = [3][stepsPerSection]byte{
	{1, 2, 0, 1, 0, 0, 3, 0, 0, 1, 0, 1, 0, 3, 0, 2}, // V0: 8 attivi
	{1, 0, 2, 1, 0, 3, 0, 0, 0, 1, 2, 0, 1, 0, 3, 0}, // V1: 8 attivi
	{2, 1, 0, 0, 1, 0, 3, 0, 0, 0, 1, 0, 2, 1, 0, 0}, // V2: 7 attivi
}

var tensionHoldRhythms = [3][stepsPerSection]byte{
	{1, 0, 1, 0, 1, 2, 0, 1, 1, 0, 1, 0, 0, 1, 0, 0}, // V0: 8 attivi (echo→pickup @5)
	{1, 0, 0, 1, 0, 1, 0, 1, 0, 1, 2, 0, 1, 0, 1, 0}, // V1: 8 attivi (echo→pickup @10)
	{1, 2, 0, 1, 0, 0, 1, 1, 0, 1, 0, 1, 0, 0, 1, 0}, // V2: 8 attivi (echo→pickup @1, trailing echo→rest @15)
}

var descReleaseRhythms = [3][stepsPerSection]byte{
	{1, 0, 0, 1, 0, 1, 0, 2, 0, 0, 1, 0, 3, 0, 0, 0}, // V0: 6 attivi (echo→pickup @7)
	{1, 0, 1, 0, 0, 1, 0, 0, 2, 0, 0, 1, 0, 3, 0, 0}, // V1: 6 attivi (echo→pickup @8)
	{1, 0, 0, 0, 1, 0, 1, 0, 0, 2, 0, 0, 1, 0, 0, 3}, // V2: 6 attivi (echo→pickup @9)
}

var rhythmsByContour = map[ContourType][3][stepsPerSection]byte{
	ContourArch:           archRhythms,
	ContourQuestionAnswer: questionAnswerRhythms,
	ContourTensionHold:    tensionHoldRhythms,
	ContourDescRelease:    descReleaseRhythms,
}

func roleFromByte(b byte) (bool, stepRole) {
	switch b {
	case 1:
		return true, roleTarget
	case 2:
		return true, rolePickup
	case 3:
		return true, roleEcho
	case 4:
		return true, roleFill
	}
	return false, ""
}

func chooseSectionContour(chordIdx int, h []byte) ContourType {
	bias := h[(chordIdx*7+3)%32] % 2
	options := [4][2]ContourType{
		{ContourArch, ContourQuestionAnswer},
		{ContourQuestionAnswer, ContourArch},
		{ContourTensionHold, ContourArch},
		{ContourDescRelease, ContourQuestionAnswer},
	}
	return options[chordIdx][bias]
}

func chooseRhythmicVariant(h []byte, sectionIdx int) int {
	return int(h[(sectionIdx*11+17)%32]) % 3
}

func buildMelodyContour(h []byte, prog theory.ChordProgression, motifDegrees []int, scaleNotes []string, key theory.Key) MelodyContour {
	var contour MelodyContour
	for i := range prog.Chords {
		ct := chooseSectionContour(i, h)
		variant := chooseRhythmicVariant(h, i)
		template := rhythmsByContour[ct][variant]
		preferHigh := i != 3

		var sec SectionContour
		sec.Type = ct
		sec.TargetDegree = motifDegrees[i%len(motifDegrees)]
		for step, code := range template {
			active, role := roleFromByte(code)
			sec.Intentions[step] = StepIntention{
				Active:     active,
				Role:       role,
				PreferHigh: active && preferHigh,
			}
		}
		contour.Sections[i] = sec
	}
	return contour
}

// withinLeap returns true if chromatic distance (with octave wrap) between a and b is <= maxSemitones.
func withinLeap(noteA, noteB string, maxSemitones int) bool {
	mA, errA := theory.NoteToMIDI(noteA + "4")
	mB, errB := theory.NoteToMIDI(noteB + "4")
	if errA != nil || errB != nil {
		return true
	}
	dist := mA - mB
	if dist < 0 {
		dist = -dist
	}
	if dist > 6 {
		dist = 12 - dist
	}
	return dist <= maxSemitones
}

// resolveTargetNote returns a chord tone close to the motif degree.
// It prefers the chord 3rd → 5th → root, picking the one nearest the scale degree,
// so roleTarget steps always count toward chord coherence.
func resolveTargetNote(chordTones, scaleNotes []string, degree int, key theory.Key) string {
	if len(scaleNotes) == 0 || len(chordTones) == 0 {
		if len(chordTones) > 0 {
			return chordTones[0]
		}
		return ""
	}
	// Always resolve to a chord tone so the validator's chord coherence check passes.
	return closestChordTone(chordTones, scaleNotes, degree)
}

// applyContourArticulation sets articulation on a target step based on ContourType and position.
func applyContourArticulation(s *schema.StepSpec, ct ContourType, pos int) {
	switch ct {
	case ContourArch:
		if pos >= 7 && pos <= 9 {
			s.Accent = true
			s.Legato = true
		} else if pos > 9 {
			s.Staccato = true
		}
	case ContourQuestionAnswer:
		if pos <= 6 {
			s.Staccato = true
		} else {
			s.Accent = true
			s.Legato = true
		}
	case ContourTensionHold:
		if pos < 8 {
			s.Legato = true
		} else {
			s.Accent = true
		}
	case ContourDescRelease:
		if pos == 0 {
			s.Accent = true
			s.Legato = true
		} else {
			s.Staccato = true
		}
	}
}

func fillMelodySection(steps []schema.StepSpec, base int, sec SectionContour, chordTones, scaleNotes []string, key theory.Key, h []byte) {
	target := resolveTargetNote(chordTones, scaleNotes, sec.TargetDegree, key)

	prevMIDI := -1
	leapDebt := 0

	for step := 0; step < stepsPerSection; step++ {
		intent := sec.Intentions[step]
		if !intent.Active {
			continue
		}

		var noteName string
		switch intent.Role {
		case rolePickup:
			noteName = approachNote(target, key)
		case roleEcho:
			echoDeg := normalizeDegree(sec.TargetDegree+1, len(scaleNotes))
			if h[(base+step)%32]%2 == 0 {
				echoDeg = normalizeDegree(sec.TargetDegree-1, len(scaleNotes))
			}
			if len(scaleNotes) > 0 {
				noteName = scaleNotes[echoDeg]
			} else {
				noteName = target
			}
		case roleFill:
			fillDeg := normalizeDegree(sec.TargetDegree+2, len(scaleNotes))
			if len(scaleNotes) > 0 {
				noteName = scaleNotes[fillDeg]
			} else {
				noteName = target
			}
		default:
			noteName = target
		}

		preferHigh := intent.PreferHigh
		if leapDebt != 0 && intent.Role == roleTarget {
			preferHigh = leapDebt > 0
			leapDebt = 0
		}

		note := melodyNote(noteName, "5", preferHigh)

		s := schema.StepSpec{Active: true, Note: note}
		switch intent.Role {
		case rolePickup:
			s.Ghost = true
			s.Staccato = true
		case roleEcho:
			s.Ghost = true
		case roleFill:
			s.Staccato = true
		default:
			applyContourArticulation(&s, sec.Type, step)
		}
		steps[base+step] = s

		currMIDI, err := theory.NoteToMIDI(note)
		if err == nil {
			if prevMIDI >= 0 {
				leap := currMIDI - prevMIDI
				if leap > 5 {
					leapDebt = -1
				} else if leap < -5 {
					leapDebt = +1
				} else {
					leapDebt = 0
				}
			}
			prevMIDI = currMIDI
		}
	}
}

func buildMelodyFromContour(steps []schema.StepSpec, contour MelodyContour, prog theory.ChordProgression, scaleNotes []string, key theory.Key, h []byte) {
	for i, chord := range prog.Chords {
		base := i * stepsPerSection
		chordTones, _ := theory.ChordNotes(chord.Root, chord.Quality)
		if len(chordTones) == 0 {
			chordTones = []string{chord.Root}
		}
		fillMelodySection(steps, base, contour.Sections[i], chordTones, scaleNotes, key, h)
	}
}

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

	steps := make([]schema.StepSpec, motifSteps)

	// Each chord section gets 16 steps (1 full musical bar) of varied bass content.
	// 4 distinct 4-step sub-patterns ensure anti-loop validator satisfaction.
	for i, chord := range prog.Chords {
		base := i * stepsPerSection
		root := chord.Root + "2"
		fifth := chordFifth(chord) + "2"
		color := bassColorTone(key, chord) + "2"

		// High root: step up to octave 3 only when MIDI stays ≤ 55
		highRoot := chord.Root + "3"
		if m, _ := theory.NoteToMIDI(highRoot); m > 55 {
			highRoot = root
		}

		approach := approachNote(chord.Root, key) + "2"

		// Select 4 distinct sub-patterns using different hash offsets per chord section.
		// Adding 2/4/6 mod 7 guarantees all four values are different (7 is prime).
		p0 := h[(i*7+2)%32] % 7
		p1 := (p0 + 2) % 7
		p2 := (p0 + 4) % 7
		p3 := (p0 + 6) % 7

		fillBassSubPattern(steps, base+0, p0, root, fifth, color, highRoot, approach, key)
		fillBassSubPattern(steps, base+4, p1, root, fifth, color, highRoot, approach, key)
		fillBassSubPattern(steps, base+8, p2, root, fifth, color, highRoot, approach, key)
		fillBassSubPattern(steps, base+12, p3, root, fifth, color, highRoot, approach, key)
	}

	applyBassOfflineStyle(style, steps, prog, h)
	if style != offlineStyleHypnotic && style != offlineStyleMinimal {
		applyPassingNotes("bassline", steps, prog, key, h)
	}
	ensureGateVariation(steps)
	ensureBassMaxDensity(steps)
	ensureBassMinDensity(steps, prog)

	return &schema.PatternSpec{
		SpecVersion:   "1.0",
		PatternType:   "bassline",
		Meta:          schema.PatternMeta{Name: "offline-bass", BPM: bpm, Key: key.Root + modeFlag(key), Bars: 4},
		Theory:        schema.TheorySpec{Key: key.Root, Mode: key.Mode, Scale: key.Scale, OctaveRange: [2]int{1, 3}},
		StyleProfile:  profileName,
		Motif:         schema.MotifSpec{Length: motifSteps, Steps: steps},
		Evolution:     buildBassEvolution(bpm, key),
		Automation:    schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: "medium"}},
		VariationSeed: seed,
	}
}

// fillBassSubPattern writes one 4-step bass groove into steps[base..base+3].
// pattern 0-6 maps to distinct rhythmic/harmonic shapes.
func fillBassSubPattern(steps []schema.StepSpec, base int, pattern byte, root, fifth, color, highRoot, approach string, key theory.Key) {
	switch pattern {
	case 0: // driving: root-color-fifth-slide
		steps[base+0] = schema.StepSpec{Active: true, Note: root, Accent: true}
		steps[base+1] = schema.StepSpec{Active: true, Note: color}
		steps[base+2] = schema.StepSpec{Active: true, Note: fifth}
		steps[base+3] = schema.StepSpec{Active: true, Note: root, Slide: true}
	case 1: // syncopated: accent-rest-fifth-ghost
		steps[base+0] = schema.StepSpec{Active: true, Note: root, Accent: true}
		steps[base+1] = schema.StepSpec{Active: false}
		steps[base+2] = schema.StepSpec{Active: true, Note: fifth}
		steps[base+3] = schema.StepSpec{Active: true, Note: root, Ghost: true}
	case 2: // octave bounce: root-highRoot-rest-slide
		steps[base+0] = schema.StepSpec{Active: true, Note: root, Accent: true}
		steps[base+1] = schema.StepSpec{Active: true, Note: highRoot}
		steps[base+2] = schema.StepSpec{Active: false}
		steps[base+3] = schema.StepSpec{Active: true, Note: root, Slide: true}
	case 3: // long sub: legato root-rest-ghost-fifth
		steps[base+0] = schema.StepSpec{Active: true, Note: root, Accent: true, Legato: true}
		steps[base+1] = schema.StepSpec{Active: false}
		steps[base+2] = schema.StepSpec{Active: true, Note: root, Ghost: true}
		steps[base+3] = schema.StepSpec{Active: true, Note: fifth, Slide: true}
	case 4: // harmonic walk: root-color-ghost root-fifth
		steps[base+0] = schema.StepSpec{Active: true, Note: root, Accent: true}
		steps[base+1] = schema.StepSpec{Active: true, Note: color}
		steps[base+2] = schema.StepSpec{Active: true, Note: root, Ghost: true}
		steps[base+3] = schema.StepSpec{Active: true, Note: fifth, Slide: true}
	case 5: // pump: rest-accent-ghost fifth-staccato root
		steps[base+0] = schema.StepSpec{Active: false}
		steps[base+1] = schema.StepSpec{Active: true, Note: root, Accent: true}
		steps[base+2] = schema.StepSpec{Active: true, Note: fifth, Ghost: true}
		steps[base+3] = schema.StepSpec{Active: true, Note: root, Staccato: true}
	default: // chromatic approach: accent-root-ghost approach-slide root
		steps[base+0] = schema.StepSpec{Active: true, Note: root, Accent: true}
		steps[base+1] = schema.StepSpec{Active: true, Note: root}
		steps[base+2] = schema.StepSpec{Active: true, Note: approach, Ghost: true}
		steps[base+3] = schema.StepSpec{Active: true, Note: root, Slide: true}
	}
}

// ensureBassMaxDensity deactivates ghost/non-accent steps when the bassline
// exceeds 13 active steps per 16-step section (the hard density limit).
func ensureBassMaxDensity(steps []schema.StepSpec) {
	const maxPerSection = 13
	sections := len(steps) / 16
	for sec := 0; sec < sections; sec++ {
		sl := steps[sec*16 : (sec+1)*16]
		for activeStepCount(sl) > maxPerSection {
			removed := false
			for i := len(sl) - 1; i >= 0; i-- {
				if sl[i].Active && sl[i].Ghost && !sl[i].Accent {
					sl[i].Active = false
					removed = true
					break
				}
			}
			if removed {
				continue
			}
			for i := len(sl) - 1; i >= 0; i-- {
				if sl[i].Active && !sl[i].Accent && !sl[i].Slide {
					sl[i].Active = false
					break
				}
			}
		}
	}
}

// ensureBassMinDensity fills inactive off-beat slots with ghost notes until
// each 16-step section has at least 8 active steps.
func ensureBassMinDensity(steps []schema.StepSpec, prog theory.ChordProgression) {
	const minPerSection = 8
	for i, chord := range prog.Chords {
		base := i * stepsPerSection
		sl := steps[base : base+stepsPerSection]
		for activeStepCount(sl) < minPerSection {
			filled := false
			for off := 2; off < stepsPerSection; off += 4 {
				if !sl[off].Active {
					sl[off] = schema.StepSpec{Active: true, Note: chord.Root + "2", Ghost: true}
					filled = true
					break
				}
			}
			if !filled {
				for off := 1; off < stepsPerSection; off += 4 {
					if !sl[off].Active {
						sl[off] = schema.StepSpec{Active: true, Note: chord.Root + "2", Ghost: true}
						break
					}
				}
				break
			}
		}
	}
}

// approachNote returns the diatonic scale degree immediately below the chord root.
func approachNote(root string, key theory.Key) string {
	scaleNotes, err := theory.ScaleNotes(key.Root, key.Scale)
	if err != nil || len(scaleNotes) == 0 {
		return root
	}
	for i, note := range scaleNotes {
		if note == root {
			if i == 0 {
				return scaleNotes[len(scaleNotes)-1]
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

// applyBassOfflineStyle modifies articulation/gate on the already-built pattern.
// It does NOT clear steps — the musical content from basslineTemplate is preserved.
func applyBassOfflineStyle(style string, steps []schema.StepSpec, prog theory.ChordProgression, h []byte) {
	switch style {
	case offlineStyleHypnotic:
		// Spacious: thin each 4-step sub-group to exactly 2 active (legato).
		// Priority order: accent > non-ghost > ghost.
		for i := 0; i < len(steps); i += 4 {
			group := steps[i : i+4]
			kept := 0
			for j := range group {
				if group[j].Active && group[j].Accent && kept < 2 {
					kept++
				} else if group[j].Active && group[j].Accent {
					group[j].Active = false
				}
			}
			for j := range group {
				if group[j].Active && !group[j].Ghost && !group[j].Accent && kept < 2 {
					kept++
				} else if group[j].Active && !group[j].Ghost && !group[j].Accent {
					group[j].Active = false
				}
			}
			for j := range group {
				if group[j].Active && group[j].Ghost && kept < 2 {
					kept++
				} else if group[j].Active && group[j].Ghost {
					group[j].Active = false
				}
			}
			for j := range group {
				if group[j].Active {
					group[j].Legato = true
					group[j].Staccato = false
					group[j].Ghost = false
				}
			}
		}
		for i := range prog.Chords {
			if steps[i*stepsPerSection].Active {
				steps[i*stepsPerSection].Accent = true
			}
		}
	case offlineStyleDriving:
		// Staccato on inner beats, occasional slide on last beat of sub-group.
		for i := range prog.Chords {
			base := i * stepsPerSection
			for sub := 0; sub < stepsPerSection; sub++ {
				pos := base + sub
				if !steps[pos].Active {
					continue
				}
				if sub%4 == 1 {
					steps[pos].Staccato = true
					steps[pos].Legato = false
				}
				if sub%4 == 3 && h[(pos)%32]%3 == 0 {
					steps[pos].Slide = true
				}
			}
		}
	case offlineStyleMinimal:
		// Thin: deactivate ghost notes on non-downbeat positions.
		for i := range steps {
			if steps[i].Ghost && i%stepsPerSection != 0 {
				steps[i].Active = false
			}
		}
	default:
		mixGateArticulation(steps, h)
	}
}

// applyArpOfflineStyle modifies articulation on the already-built arp pattern.
func applyArpOfflineStyle(style string, steps []schema.StepSpec, prog theory.ChordProgression, h []byte) {
	switch style {
	case offlineStyleHypnotic:
		// Full legato on all active notes; section downbeats accented.
		for i := range steps {
			if steps[i].Active {
				steps[i].Legato = true
				steps[i].Staccato = false
			}
		}
		for i := range prog.Chords {
			if steps[i*stepsPerSection].Active {
				steps[i*stepsPerSection].Accent = true
			}
		}
	case offlineStyleDriving:
		// Alternate legato/staccato pairs within each 4-step sub-group.
		for i := range steps {
			if !steps[i].Active {
				continue
			}
			switch (i / 2) % 2 {
			case 0:
				steps[i].Legato = true
				steps[i].Staccato = false
			case 1:
				steps[i].Staccato = true
				steps[i].Legato = false
			}
		}
	case offlineStyleMinimal:
		// Arpeggio needs high density (≥48/64), so minimal = quiet staccato articulation.
		// Section downbeats stay legato to ensure gate variation (required by validator).
		for i := range steps {
			if steps[i].Active {
				steps[i].Staccato = true
				steps[i].Legato = false
				steps[i].Accent = false
				steps[i].Ghost = false
			}
		}
		for i := range prog.Chords {
			if steps[i*stepsPerSection].Active {
				steps[i*stepsPerSection].Legato = true
				steps[i*stepsPerSection].Staccato = false
			}
		}
	default:
		mixGateArticulation(steps, h)
	}
}

// applyMelodyOfflineStyle modifies articulation on the already-built melody pattern.
func applyMelodyOfflineStyle(style string, steps []schema.StepSpec, prog theory.ChordProgression, key theory.Key, h []byte, scaleNotes []string) {
	switch style {
	case offlineStyleHypnotic:
		// Legato on all active notes; remove ghost notes for spaciousness.
		for i := range steps {
			if steps[i].Active {
				if steps[i].Ghost {
					steps[i].Active = false
				} else {
					steps[i].Legato = true
					steps[i].Staccato = false
				}
			}
		}
	case offlineStyleDriving:
		// Staccato on non-accent, non-legato notes.
		for i := range steps {
			if steps[i].Active && !steps[i].Accent && !steps[i].Legato {
				steps[i].Staccato = true
			}
		}
	case offlineStyleMinimal:
		// Minimal: place exactly 4 chord tones per section at the 4 downbeat positions.
		// This preserves harmonic identity while meeting the minimum density (4×4=16/64).
		for sec, chord := range prog.Chords {
			base := sec * stepsPerSection
			for i := 0; i < stepsPerSection; i++ {
				steps[base+i].Active = false
			}
			notes := chordNotesOrRoot(chord)
			n := len(notes)
			steps[base+0] = schema.StepSpec{Active: true, Note: melodyNote(notes[0], "5", false), Accent: true, Legato: true}
			steps[base+4] = schema.StepSpec{Active: true, Note: melodyNote(notes[1%n], "5", false), Staccato: true}
			steps[base+8] = schema.StepSpec{Active: true, Note: melodyNote(notes[2%n], "5", false), Legato: true}
			steps[base+12] = schema.StepSpec{Active: true, Note: melodyNote(notes[0], "5", false), Ghost: true}
		}
	default:
		mixGateArticulation(steps, h)
	}
}

func applyPassingNotes(patternType string, steps []schema.StepSpec, prog theory.ChordProgression, key theory.Key, h []byte) {
	maxActive := map[string]int{"bassline": 13 * (len(steps) / 16), "melody": 8 * (len(steps) / 16)}[patternType]
	if maxActive == 0 {
		return
	}
	chance := 15 + int(h[20])%11
	for i := range prog.Chords {
		if activeStepCount(steps) >= maxActive || int(h[(21+i)%len(h)])%100 >= chance {
			continue
		}
		base := i * stepsPerSection
		for pos := base + 1; pos <= base+stepsPerSection-1 && pos < len(steps); pos++ {
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
	const minPerSection = 4
	for i, chord := range prog.Chords {
		base := i * stepsPerSection
		sl := steps[base : base+stepsPerSection]
		if activeStepCount(sl) >= minPerSection {
			continue
		}
		notes := chordNotesOrRoot(chord)
		sl[0] = schema.StepSpec{Active: true, Note: notes[0] + "5", Accent: i == 2, Legato: true}
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

	steps := make([]schema.StepSpec, motifSteps)

	// Open voicing technique: root in low octave, upper voices spread wide.
	voicingStyle := h[5] % 4
	scaleNotes, _ := theory.ScaleNotes(key.Root, key.Scale)
	if len(scaleNotes) < 7 {
		scaleNotes = []string{"C", "D", "E", "F", "G", "A", "B"}
	}

	for i, chord := range prog.Chords {
		notes, err := theory.ChordNotes(chord.Root, chord.Quality)
		if err != nil || len(notes) < 3 {
			notes = []string{chord.Root, chord.Root, chord.Root}
		}

		// Extended 4-note pool including a scale extension (7th or 9th).
		pool := buildArpPool(notes, scaleNotes, voicingStyle)

		base := i * stepsPerSection

		// 4 sub-groups with 4 different arp patterns for the chord.
		p0 := chooseArpPattern(h, key, i*4+0)
		p1 := chooseArpPattern(h, key, i*4+1)
		p2 := chooseArpPattern(h, key, i*4+2)
		p3 := chooseArpPattern(h, key, i*4+3)
		// Guarantee all 4 are distinct within the chord section.
		p1 = ensureDistinct(p1, []byte{p0}, 6)
		p2 = ensureDistinct(p2, []byte{p0, p1}, 6)
		p3 = ensureDistinct(p3, []byte{p0, p1, p2}, 6)

		fillArpSubPattern(steps, base+0, p0, pool)
		fillArpSubPattern(steps, base+4, p1, pool)
		fillArpSubPattern(steps, base+8, p2, pool)
		fillArpSubPattern(steps, base+12, p3, pool)
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
		Meta:          schema.PatternMeta{Name: "offline-arp", BPM: bpm, Key: key.Root + modeFlag(key), Bars: 4},
		Theory:        schema.TheorySpec{Key: key.Root, Mode: key.Mode, Scale: key.Scale, OctaveRange: [2]int{3, 6}},
		StyleProfile:  profileName,
		Motif:         schema.MotifSpec{Length: motifSteps, Steps: steps},
		Evolution:     buildArpEvolution(bpm, key),
		Automation:    schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: sweepStyle}},
		VariationSeed: seed,
	}
}

// buildArpPool returns a 4-note pool using open voicing plus a scale extension (7th or 9th).
func buildArpPool(notes []string, scaleNotes []string, voicingStyle byte) []string {
	// Scale extension: 7th degree (index 6) adds color; 2nd degree (index 1) adds openness.
	ext := scaleNotes[6] // 7th
	if voicingStyle%2 == 1 {
		ext = scaleNotes[1] // 9th (2nd)
	}
	extNote := arpClamp(ext + "5")

	// pool[0-2] are always chord tones; pool[3] is always the extension.
	// This invariant is critical for chord-coherence validation (80% threshold requires ≤2 ext/16 steps).
	switch voicingStyle % 4 {
	case 0: // wide open: root3, 5th4, 3rd5, ext5
		return []string{arpClamp(notes[0] + "3"), arpClamp(notes[2] + "4"), arpClamp(notes[1] + "5"), extNote}
	case 1: // 1st inversion: 3rd3, 5th4, root5, ext5
		return []string{arpClamp(notes[1] + "3"), arpClamp(notes[2] + "4"), arpClamp(notes[0] + "5"), extNote}
	case 2: // 2nd inversion: 5th3, root4, 3rd5, ext5
		return []string{arpClamp(notes[2] + "3"), arpClamp(notes[0] + "4"), arpClamp(notes[1] + "5"), extNote}
	default: // drop-2: root3, 3rd4, 5th4, ext5
		return []string{arpClamp(notes[0] + "3"), arpClamp(notes[1] + "4"), arpClamp(notes[2] + "4"), extNote}
	}
}

// arpClamp returns the note clamped to the arp MIDI range [48, 84].
func arpClamp(note string) string {
	name := theory.NoteNameOnly(note)
	octave := noteOctave(note)
	full := name + octave
	midi, err := theory.NoteToMIDI(full)
	if err != nil {
		return full
	}
	if midi < 48 {
		return name + "4"
	}
	if midi > 84 {
		return name + "4"
	}
	return full
}

// fillArpSubPattern writes one 4-step arp pattern into steps[base..base+3].
// pool[3] is a scale extension (7th/9th, not a chord tone); it appears only in
// cases 0 and 4 to keep chord-coherence ≥ 80% across a 16-step section.
func fillArpSubPattern(steps []schema.StepSpec, base int, pattern byte, pool []string) {
	switch pattern % 6 {
	case 0: // ascending with extension: root → 3rd → 5th → ext
		steps[base+0] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
		steps[base+1] = schema.StepSpec{Active: true, Note: pool[1], Staccato: true}
		steps[base+2] = schema.StepSpec{Active: true, Note: pool[2]}
		steps[base+3] = schema.StepSpec{Active: true, Note: pool[3], Staccato: true}
	case 1: // descending chord tones: 5th → 3rd → root → 3rd
		steps[base+0] = schema.StepSpec{Active: true, Note: pool[2], Accent: true}
		steps[base+1] = schema.StepSpec{Active: true, Note: pool[1]}
		steps[base+2] = schema.StepSpec{Active: true, Note: pool[0], Staccato: true}
		steps[base+3] = schema.StepSpec{Active: true, Note: pool[1]}
	case 2: // pendulum chord tones: root-5th-3rd-root
		steps[base+0] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
		steps[base+1] = schema.StepSpec{Active: true, Note: pool[2], Staccato: true}
		steps[base+2] = schema.StepSpec{Active: true, Note: pool[1]}
		steps[base+3] = schema.StepSpec{Active: true, Note: pool[0], Staccato: true}
	case 3: // octave pulse chord tones: root-5th alternating (no extension)
		steps[base+0] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
		steps[base+1] = schema.StepSpec{Active: true, Note: pool[2], Staccato: true}
		steps[base+2] = schema.StepSpec{Active: true, Note: pool[0]}
		steps[base+3] = schema.StepSpec{Active: true, Note: pool[2], Accent: true}
	case 4: // broken with extension: root-5th-root-ext (extension at tail only)
		steps[base+0] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
		steps[base+1] = schema.StepSpec{Active: true, Note: pool[2], Staccato: true}
		steps[base+2] = schema.StepSpec{Active: true, Note: pool[0]}
		steps[base+3] = schema.StepSpec{Active: true, Note: pool[3], Legato: true}
	default: // climbing chord tones: root-3rd-5th-3rd (no extension)
		steps[base+0] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
		steps[base+1] = schema.StepSpec{Active: true, Note: pool[1], Staccato: true}
		steps[base+2] = schema.StepSpec{Active: true, Note: pool[2]}
		steps[base+3] = schema.StepSpec{Active: true, Note: pool[1], Staccato: true}
	}
}

// ensureDistinct increments p until it is not in used, cycling mod maxVal.
func ensureDistinct(p byte, used []byte, maxVal byte) byte {
	for {
		found := false
		for _, u := range used {
			if p == u {
				found = true
				break
			}
		}
		if !found {
			return p
		}
		p = (p + 1) % maxVal
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

	steps := make([]schema.StepSpec, motifSteps)

	// Short motif (3-5 scale degrees) seeded from hash for variety.
	motifLen := 3 + int(h[6])%3
	motifDegrees := buildHypnoticMotif(h, motifLen, scaleNotes)

	// Build a full melodic contour across all 4 sections, then render it.
	// The phrase builder assigns per-section rhythmic variants and note shapes
	// driven by contour type (arch, descent, question_answer, wave).
	contour := buildMelodyContour(h, prog, motifDegrees, scaleNotes, key)
	buildMelodyFromContour(steps, contour, prog, scaleNotes, key, h)

	applyMelodyOfflineStyle(style, steps, prog, key, h, scaleNotes)
	if style != offlineStyleHypnotic && style != offlineStyleMinimal {
		applyPassingNotes("melody", steps, prog, key, h)
	}
	ensureMelodyMinDensity(steps, prog)
	ensureGateVariation(steps)

	return &schema.PatternSpec{
		SpecVersion:   "1.0",
		PatternType:   "melody",
		Meta:          schema.PatternMeta{Name: "offline-melody", BPM: bpm, Key: key.Root + modeFlag(key), Bars: 4},
		Theory:        schema.TheorySpec{Key: key.Root, Mode: key.Mode, Scale: key.Scale, OctaveRange: [2]int{4, 6}},
		StyleProfile:  profileName,
		Motif:         schema.MotifSpec{Length: motifSteps, Steps: steps},
		Evolution:     buildMelodyEvolution(bpm, key),
		Automation:    schema.AutomationIntent{ModWheel: &schema.ModWheelIntent{Style: "subtle"}},
		VariationSeed: seed,
	}
}

// buildMelodyBar fills 16 steps (1 full musical bar) for one chord section.
// chordIdx 0=statement, 1=call, 2=tension, 3=resolution — each with distinct density and phrasing.
func buildMelodyBar(steps []schema.StepSpec, base, chordIdx int, chord theory.ProgressionChord, scaleNotes []string, motifDegrees []int, key theory.Key, h []byte) {
	chordTones, _ := theory.ChordNotes(chord.Root, chord.Quality)
	if len(chordTones) == 0 {
		chordTones = []string{chord.Root}
	}

	degree := motifDegrees[chordIdx%len(motifDegrees)]
	target := closestChordTone(chordTones, scaleNotes, degree)
	targetNote := melodyNote(target, "5", false)

	pickup := melodyNote(approachNote(target, key), "5", false)
	charDeg := characterDegree(key)
	charNote := melodyNote(scaleNotes[normalizeDegree(charDeg, len(scaleNotes))], "5", false)
	echoNote := melodyNote(scaleNotes[normalizeDegree(degree+2, len(scaleNotes))], "5", false)
	highNote := melodyNote(target, "6", false)

	switch chordIdx {
	case 0: // Statement: sparse, held note establishes the phrase identity
		steps[base+0] = schema.StepSpec{Active: true, Note: targetNote, Accent: true, Legato: true}
		if h[(base+2)%32]%3 != 0 {
			answerDeg := normalizeDegree(degree+4, len(scaleNotes))
			steps[base+2] = schema.StepSpec{Active: true, Note: melodyNote(scaleNotes[answerDeg], "5", false), Staccato: true}
		}
		steps[base+5] = schema.StepSpec{Active: true, Note: targetNote, Ghost: true}
		if h[(base+9)%32]%2 == 0 {
			steps[base+9] = schema.StepSpec{Active: true, Note: echoNote, Staccato: true}
		}
		steps[base+14] = schema.StepSpec{Active: true, Note: pickup, Ghost: true, Staccato: true}

	case 1: // Call: pickup→landing→development phrase, more active than statement
		steps[base+0] = schema.StepSpec{Active: true, Note: pickup, Ghost: true, Staccato: true}
		steps[base+1] = schema.StepSpec{Active: true, Note: targetNote, Accent: true, Legato: true}
		steps[base+4] = schema.StepSpec{Active: true, Note: echoNote, Staccato: true}
		if h[(base+6)%32]%2 == 0 {
			steps[base+6] = schema.StepSpec{Active: true, Note: targetNote, Ghost: true}
		}
		steps[base+8] = schema.StepSpec{Active: true, Note: melodyNote(scaleNotes[normalizeDegree(degree+5, len(scaleNotes))], "5", false), Staccato: true}
		if h[(base+12)%32]%3 != 0 {
			steps[base+12] = schema.StepSpec{Active: true, Note: targetNote, Ghost: true}
		}
		steps[base+14] = schema.StepSpec{Active: true, Note: charNote, Staccato: true}

	case 2: // Tension: dense, modal character note at peak, syncopated off-beats
		steps[base+0] = schema.StepSpec{Active: true, Note: charNote, Accent: true}
		steps[base+2] = schema.StepSpec{Active: true, Note: melodyNote(scaleNotes[normalizeDegree(charDeg+1, len(scaleNotes))], "5", false), Staccato: true}
		steps[base+4] = schema.StepSpec{Active: true, Note: targetNote, Ghost: true}
		if h[(base+5)%32]%2 == 0 {
			steps[base+5] = schema.StepSpec{Active: true, Note: charNote, Staccato: true}
		}
		steps[base+7] = schema.StepSpec{Active: true, Note: highNote, Staccato: true}
		steps[base+9] = schema.StepSpec{Active: true, Note: charNote, Ghost: true}
		if h[(base+11)%32]%2 == 0 {
			steps[base+11] = schema.StepSpec{Active: true, Note: melodyNote(scaleNotes[normalizeDegree(charDeg-1, len(scaleNotes))], "5", false), Staccato: true}
		}
		steps[base+13] = schema.StepSpec{Active: true, Note: targetNote, Staccato: true}

	case 3: // Resolution: descend back home, exhale, closing gesture
		steps[base+0] = schema.StepSpec{Active: true, Note: targetNote, Accent: true, Legato: true}
		steps[base+3] = schema.StepSpec{Active: true, Note: melodyNote(scaleNotes[normalizeDegree(degree-1, len(scaleNotes))], "5", false), Staccato: true}
		steps[base+7] = schema.StepSpec{Active: true, Note: melodyNote(scaleNotes[normalizeDegree(degree-2, len(scaleNotes))], "4", false), Ghost: true}
		if h[(base+9)%32]%3 != 2 {
			steps[base+9] = schema.StepSpec{Active: true, Note: melodyNote(scaleNotes[normalizeDegree(degree-3, len(scaleNotes))], "4", false), Staccato: true}
		}
		steps[base+12] = schema.StepSpec{Active: true, Note: targetNote, Ghost: true}
	}
}

// melodyNote returns noteName+preferredOct clamped to the valid melody range [60, 84].
// When preferHigher is true and preferredOct is "5", the octave is raised to "6" first.
func melodyNote(noteName, preferredOct string, preferHigher bool) string {
	oct := preferredOct
	if preferHigher && oct == "5" {
		oct = "6"
	}
	note := noteName + oct
	midi, err := theory.NoteToMIDI(note)
	if err != nil {
		return noteName + preferredOct
	}
	if midi < 60 {
		return noteName + "5"
	}
	if midi > 84 {
		return noteName + "4"
	}
	return note
}

// buildMelodyPhraseSection is kept for backward compatibility (used by tests).
func buildMelodyPhraseSection(steps []schema.StepSpec, base, sectionIdx int, chord theory.ProgressionChord, scaleNotes []string, motifDegrees []int, key theory.Key, h []byte) {
	chordTones, _ := theory.ChordNotes(chord.Root, chord.Quality)
	if len(chordTones) == 0 {
		chordTones = []string{chord.Root}
	}

	degree := motifDegrees[sectionIdx%len(motifDegrees)]
	target := closestChordTone(chordTones, scaleNotes, degree)
	targetNote := melodyNote(target, "5", false)

	switch sectionIdx % 4 {
	case 0:
		steps[base] = schema.StepSpec{Active: true, Note: targetNote, Accent: true, Legato: true}
		if h[base%32]%3 == 0 {
			answerDeg := normalizeDegree(degree+4, len(scaleNotes))
			steps[base+2] = schema.StepSpec{Active: true, Note: melodyNote(scaleNotes[answerDeg], "5", false), Staccato: true}
		}
	case 1:
		pickup := approachNote(target, key)
		steps[base] = schema.StepSpec{Active: true, Note: melodyNote(pickup, "5", false), Ghost: true, Staccato: true}
		steps[base+1] = schema.StepSpec{Active: true, Note: targetNote, Accent: true, Legato: true}
		if h[(base+3)%32]%2 == 0 {
			echoDeg := normalizeDegree(degree+2, len(scaleNotes))
			steps[base+3] = schema.StepSpec{Active: true, Note: melodyNote(scaleNotes[echoDeg], "5", false), Staccato: true}
		}
	case 2:
		charDeg := characterDegree(key)
		charName := scaleNotes[normalizeDegree(charDeg, len(scaleNotes))]
		steps[base] = schema.StepSpec{Active: true, Note: melodyNote(charName, "5", false), Accent: true}
		tensionDeg := normalizeDegree(charDeg+1, len(scaleNotes))
		steps[base+2] = schema.StepSpec{Active: true, Note: melodyNote(scaleNotes[tensionDeg], "5", false), Staccato: true}
		if h[(base+3)%32]%2 == 0 {
			steps[base+3] = schema.StepSpec{Active: true, Note: targetNote, Ghost: true}
		}
	case 3:
		steps[base] = schema.StepSpec{Active: true, Note: targetNote, Accent: true, Legato: true}
		if h[(base+2)%32]%3 != 2 {
			descDeg := normalizeDegree(degree-1, len(scaleNotes))
			steps[base+2] = schema.StepSpec{Active: true, Note: melodyNote(scaleNotes[descDeg], "4", false), Staccato: true}
		}
	}
}

// buildHypnoticMotif creates a short motif using wide intervals for an open, hypnotic sound.
func buildHypnoticMotif(h []byte, length int, scaleNotes []string) []int {
	numDegrees := len(scaleNotes)
	motif := make([]int, length)

	startDegrees := []int{0, 3, 4}
	motif[0] = startDegrees[int(h[14])%len(startDegrees)]

	wideIntervals := []int{3, 4, 5, -3, -4, -2}
	for i := 1; i < length; i++ {
		interval := wideIntervals[int(h[15+i])%len(wideIntervals)]

		next := (motif[i-1] + interval) % numDegrees
		if next < 0 {
			next += numDegrees
		}

		// Contrary motion guard: prevent two consecutive stored-degree-diffs
		// in the same direction both >= 3. Compare actual stored diffs, not
		// raw intervals, to handle modulo-wrap correctly.
		if i > 1 {
			prevDiff := motif[i-1] - motif[i-2]
			storedDiff := next - motif[i-1]
			absPrev := prevDiff
			if absPrev < 0 {
				absPrev = -absPrev
			}
			absStored := storedDiff
			if absStored < 0 {
				absStored = -absStored
			}
			sameDir := (prevDiff > 0 && storedDiff > 0) || (prevDiff < 0 && storedDiff < 0)
			if sameDir && absPrev >= 3 && absStored >= 3 {
				altNext := (motif[i-1] - interval) % numDegrees
				if altNext < 0 {
					altNext += numDegrees
				}
				altDiff := altNext - motif[i-1]
				absAlt := altDiff
				if absAlt < 0 {
					absAlt = -absAlt
				}
				altSameDir := (prevDiff > 0 && altDiff > 0) || (prevDiff < 0 && altDiff < 0)
				if !altSameDir || absAlt < 3 {
					next = altNext
				}
			}
		}

		motif[i] = next
	}

	return motif
}

func chooseArpPattern(h []byte, key theory.Key, section int) byte {
	characterBias := byte(characterDegree(key) + len(key.Root))
	return (h[(section+4)%32] + characterBias + byte(section)) % 6
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
		return 5
	case "dorian":
		return 5
	case "phrygian":
		return 1
	case "mixolydian":
		return 6
	case "lydian":
		return 3
	default:
		return 4
	}
}

// buildBassEvolution: 4 renderer iterations map to introduce/build/peak/release.
func buildBassEvolution(bpm float64, key theory.Key) []schema.EvolutionStep {
	if bpm >= 128 {
		return []schema.EvolutionStep{
			{FromBar: 1, ToBar: 1, Action: "build", Intensity: 0.7},
			{FromBar: 2, ToBar: 2, Action: "build", Intensity: 0.85},
			{FromBar: 3, ToBar: 3, Action: "peak", Intensity: 1.0},
			{FromBar: 4, ToBar: 4, Action: "release", Intensity: 0.6},
		}
	}
	return []schema.EvolutionStep{
		{FromBar: 1, ToBar: 1, Action: "introduce", Intensity: 0.5},
		{FromBar: 2, ToBar: 2, Action: "build", Intensity: 0.7},
		{FromBar: 3, ToBar: 3, Action: "peak", Intensity: 0.9},
		{FromBar: 4, ToBar: 4, Action: "release", Intensity: 0.5},
	}
}

// buildArpEvolution: arp enters at full density and peaks mid-track.
func buildArpEvolution(bpm float64, key theory.Key) []schema.EvolutionStep {
	return []schema.EvolutionStep{
		{FromBar: 1, ToBar: 1, Action: "build", Intensity: 0.7},
		{FromBar: 2, ToBar: 2, Action: "peak", Intensity: 0.9},
		{FromBar: 3, ToBar: 3, Action: "peak", Intensity: 1.0},
		{FromBar: 4, ToBar: 4, Action: "release", Intensity: 0.7},
	}
}

// buildMelodyEvolution: melody builds from sparse intro to phrase peak.
func buildMelodyEvolution(bpm float64, key theory.Key) []schema.EvolutionStep {
	return []schema.EvolutionStep{
		{FromBar: 1, ToBar: 1, Action: "introduce", Intensity: 0.4},
		{FromBar: 2, ToBar: 2, Action: "build", Intensity: 0.6},
		{FromBar: 3, ToBar: 3, Action: "peak", Intensity: 0.8},
		{FromBar: 4, ToBar: 4, Action: "release", Intensity: 0.4},
	}
}

func chooseBassProfile(key theory.Key, bpm float64, h []byte) string {
	if bpm < 118 {
		candidates := []string{"bass_sub", "bass_progressive"}
		return candidates[int(h[0])%len(candidates)]
	}
	if bpm >= 128 {
		candidates := []string{"bass_driving", "bass_progressive"}
		if key.Mode == "minor" {
			return "bass_driving"
		}
		return candidates[int(h[0])%len(candidates)]
	}
	candidates := []string{"bass_progressive"}
	if key.Mode == "minor" {
		candidates = append(candidates, "bass_driving")
	}
	return candidates[int(h[0])%len(candidates)]
}

func chooseArpProfile(key theory.Key, bpm float64, h []byte) string {
	if bpm < 116 {
		return "arp_flowing"
	}
	if bpm >= 130 {
		candidates := []string{"arp_staccato", "arp_epic"}
		return candidates[int(h[1])%len(candidates)]
	}
	if key.Mode == "minor" {
		if bpm >= 126 {
			candidates := []string{"arp_epic", "arp_staccato"}
			return candidates[int(h[1])%len(candidates)]
		}
		candidates := []string{"arp_flowing", "arp_epic"}
		return candidates[int(h[1])%len(candidates)]
	}
	candidates := []string{"arp_flowing", "arp_epic"}
	return candidates[int(h[1])%len(candidates)]
}

func chooseMelodyProfile(key theory.Key, bpm float64, h []byte) string {
	if bpm < 118 {
		return "melody_expressive"
	}
	if key.Mode == "minor" {
		return "melody_hypnotic"
	}
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
