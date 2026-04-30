package generator

import (
	"crypto/sha256"
	"math"

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

// OfflineTemplate is the public version of offlineTemplate for use by service layer.
func OfflineTemplate(patternType string, musicCtx MusicContext) *schema.PatternSpec {
	return offlineTemplate(patternType, musicCtx)
}

func basslineTemplate(ctx MusicContext) *schema.PatternSpec {
	key := ctx.Key
	prog := ctx.ChordProgression
	seed := ctx.VariationSeed
	bpm := ctx.BPM

	h := seedHash(seed + "bass")
	profileName := chooseBassProfile(key, bpm, h)

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

		switch pattern {
		case 0: // driving 16ths: root-root-fifth-slide
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: root + "2"}
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
		case 4: // rolling triplet feel: root-fifth-root-fifth (tight)
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: fifth + "2"}
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

// approachNote returns a chromatic approach note one semitone below the target.
func approachNote(root string, _ theory.Key) string {
	midi, err := theory.NoteToMIDI(root + "3")
	if err != nil || midi <= 0 {
		return root
	}
	note := theory.MIDIToNote(midi - 1)
	if len(note) > 0 && note[len(note)-1] >= '0' && note[len(note)-1] <= '9' {
		note = note[:len(note)-1]
	}
	return note
}

func arpeggioTemplate(ctx MusicContext) *schema.PatternSpec {
	key := ctx.Key
	prog := ctx.ChordProgression
	seed := ctx.VariationSeed
	bpm := ctx.BPM

	h := seedHash(seed + "arp")
	profileName := chooseArpProfile(key, bpm, h)

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

		arpPattern := h[(i+4)%32] % 6
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

	h := seedHash(seed + "melody")
	profileName := chooseMelodyProfile(key, bpm, h)

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

		degree := motifDegrees[motifNoteIdx%len(motifDegrees)]
		noteName := scaleNotes[degree%len(scaleNotes)]

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

func buildEvolution(bpm float64, key theory.Key) []schema.EvolutionStep {
	if key.Mode == "minor" && bpm >= 124 {
		return []schema.EvolutionStep{
			{FromBar: 1, ToBar: 4, Action: "introduce", Intensity: 0.4},
			{FromBar: 5, ToBar: 8, Action: "build", Intensity: 0.6},
			{FromBar: 9, ToBar: 12, Action: "peak", Intensity: 0.9},
			{FromBar: 13, ToBar: 16, Action: "release", Intensity: 0.5},
		}
	}
	return []schema.EvolutionStep{
		{FromBar: 1, ToBar: 4, Action: "introduce", Intensity: 0.3},
		{FromBar: 5, ToBar: 8, Action: "build", Intensity: 0.5},
		{FromBar: 9, ToBar: 12, Action: "peak", Intensity: 0.8},
		{FromBar: 13, ToBar: 16, Action: "release", Intensity: 0.6},
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
		candidates := []string{"bass_techno_driving", "bass_progressive"}
		if key.Mode == "minor" {
			// Minor keys lean toward darker driving sound
			return "bass_techno_driving"
		}
		return candidates[int(h[0])%len(candidates)]
	}
	// Mid-range 118-127 — progressive house territory
	candidates := []string{"bass_progressive"}
	if key.Mode == "minor" {
		candidates = append(candidates, "bass_techno_driving")
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
