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

func basslineTemplate(ctx MusicContext) *schema.PatternSpec {
	key := ctx.Key
	prog := ctx.ChordProgression
	seed := ctx.VariationSeed
	bpm := ctx.BPM

	h := seedHash(seed + "bass")
	profileName := chooseBassProfile(key, bpm, h)

	steps := make([]schema.StepSpec, 16)
	for i, chord := range prog.Chords {
		root := chord.Root
		fifth := chordFifth(chord)
		base := i * 4

		pattern := h[i%32] % 6
		switch pattern {
		case 0: // driving: root-root-rest-slide
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: root + "2"}
			steps[base+2] = schema.StepSpec{Active: false}
			steps[base+3] = schema.StepSpec{Active: true, Note: fifth + "2", Slide: true}
		case 1: // syncopated: root-rest-fifth-ghost
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: false}
			steps[base+2] = schema.StepSpec{Active: true, Note: fifth + "2"}
			steps[base+3] = schema.StepSpec{Active: true, Note: root + "2", Ghost: true}
		case 2: // octave jump: root-highRoot-rest-slide
			highNote := root + "3"
			if midi, _ := theory.NoteToMIDI(highNote); midi > 55 {
				highNote = root + "2"
			}
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: highNote}
			steps[base+2] = schema.StepSpec{Active: false}
			steps[base+3] = schema.StepSpec{Active: true, Note: root + "2", Slide: true}
		case 3: // minimal: root-rest-rest-ghost
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: false}
			steps[base+2] = schema.StepSpec{Active: false}
			steps[base+3] = schema.StepSpec{Active: true, Note: root + "2", Ghost: true}
		case 4: // rolling: root-fifth-root-slide
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: fifth + "2"}
			steps[base+2] = schema.StepSpec{Active: true, Note: root + "2"}
			steps[base+3] = schema.StepSpec{Active: true, Note: fifth + "2", Slide: true}
		default: // pulsing: root-root-root-rest
			steps[base] = schema.StepSpec{Active: true, Note: root + "2", Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: root + "2"}
			steps[base+2] = schema.StepSpec{Active: true, Note: root + "2", Ghost: true}
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
		Evolution:     buildEvolution(bpm, key),
		Automation:    schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: "medium"}},
		VariationSeed: seed,
	}
}

func arpeggioTemplate(ctx MusicContext) *schema.PatternSpec {
	key := ctx.Key
	prog := ctx.ChordProgression
	seed := ctx.VariationSeed
	bpm := ctx.BPM

	h := seedHash(seed + "arp")
	profileName := chooseArpProfile(key, bpm, h)

	steps := make([]schema.StepSpec, 16)
	prevPattern := byte(255)
	for i, chord := range prog.Chords {
		notes, err := theory.ChordNotes(chord.Root, chord.Quality)
		if err != nil || len(notes) < 3 {
			notes = []string{chord.Root, chord.Root, chord.Root}
		}

		pool := []string{notes[0] + "3", notes[1] + "4", notes[2] + "4", notes[0] + "4"}
		base := i * 4

		arpPattern := h[(i+4)%32] % 5
		if i > 0 && arpPattern == prevPattern {
			arpPattern = (arpPattern + 1) % 5
		}
		prevPattern = arpPattern
		switch arpPattern {
		case 0: // ascending
			steps[base] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: pool[1], Staccato: true}
			steps[base+2] = schema.StepSpec{Active: true, Note: pool[2], Staccato: true}
			steps[base+3] = schema.StepSpec{Active: true, Note: pool[3], Staccato: true}
		case 1: // descending
			steps[base] = schema.StepSpec{Active: true, Note: pool[3], Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: pool[2], Staccato: true}
			steps[base+2] = schema.StepSpec{Active: true, Note: pool[1], Staccato: true}
			steps[base+3] = schema.StepSpec{Active: true, Note: pool[0], Staccato: true}
		case 2: // pendulum
			steps[base] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: pool[2], Staccato: true}
			steps[base+2] = schema.StepSpec{Active: true, Note: pool[1], Staccato: true}
			steps[base+3] = schema.StepSpec{Active: true, Note: pool[3], Staccato: true}
		case 3: // skip
			steps[base] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: pool[3], Staccato: true}
			steps[base+2] = schema.StepSpec{Active: true, Note: pool[1], Staccato: true}
			steps[base+3] = schema.StepSpec{Active: false}
		default: // pulse
			steps[base] = schema.StepSpec{Active: true, Note: pool[0], Accent: true}
			steps[base+1] = schema.StepSpec{Active: true, Note: pool[1], Staccato: true}
			steps[base+2] = schema.StepSpec{Active: true, Note: pool[0], Staccato: true}
			steps[base+3] = schema.StepSpec{Active: true, Note: pool[2], Staccato: true}
		}
	}

	return &schema.PatternSpec{
		SpecVersion:   "1.0",
		PatternType:   "arpeggio",
		Meta:          schema.PatternMeta{Name: "offline-arp", BPM: bpm, Key: key.Root + modeFlag(key), Bars: 16},
		Theory:        schema.TheorySpec{Key: key.Root, Mode: key.Mode, Scale: key.Scale, OctaveRange: [2]int{3, 6}},
		StyleProfile:  profileName,
		Motif:         schema.MotifSpec{Length: 16, Steps: steps},
		Evolution:     buildEvolution(bpm, key),
		Automation:    schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: "subtle"}},
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

	contours := [][]int{
		{4, 3, 2, 4, 5, 3, 2, 0},    // arch: rise-fall
		{6, 5, 4, 3, 4, 5, 6, 4},    // wave: descending oscillation
		{0, 2, 4, 3, 5, 4, 2, 0},    // call-response
		{4, 4, 5, 6, 4, 3, 2, 3},    // peak-centered
		{2, 3, 4, 5, 6, 5, 4, 2},    // mountain
	}
	contourIdx := int(h[8]) % len(contours)
	contour := contours[contourIdx]

	rhythms := [][]bool{
		{true, true, false, false, true, false, true, true, false, false, false, true, true, false, true, false},
		{true, false, true, false, true, true, false, false, true, false, false, true, false, true, false, false},
		{true, true, false, true, false, false, true, false, false, true, false, false, true, true, false, false},
		{true, false, false, true, true, false, true, false, true, false, false, false, true, false, true, false},
	}
	rhythmIdx := int(h[12]) % len(rhythms)
	rhythm := rhythms[rhythmIdx]

	noteIdx := 0
	for i := 0; i < 16; i++ {
		if !rhythm[i] {
			steps[i] = schema.StepSpec{Active: false}
			continue
		}

		chordIdx := i / 4
		chord := prog.Chords[chordIdx]
		chordTones, _ := theory.ChordNotes(chord.Root, chord.Quality)

		degree := contour[noteIdx%len(contour)]
		noteName := scaleNotes[degree%len(scaleNotes)]

		if noteIdx%3 == 0 && len(chordTones) > 0 {
			noteName = closestChordTone(chordTones, scaleNotes, degree)
		}

		octave := "5"
		if degree >= 5 {
			octave = "5"
		} else if degree <= 1 {
			octave = "4"
		}

		midiVal, _ := theory.NoteToMIDI(noteName + octave)
		if midiVal < 60 {
			octave = "5"
		} else if midiVal > 96 {
			octave = "4"
		}

		isAccent := i%4 == 0
		isLegato := noteIdx > 0 && noteIdx%2 == 0

		steps[i] = schema.StepSpec{
			Active: true,
			Note:   noteName + octave,
			Accent: isAccent,
			Legato: isLegato,
		}
		noteIdx++
	}

	return &schema.PatternSpec{
		SpecVersion:   "1.0",
		PatternType:   "melody",
		Meta:          schema.PatternMeta{Name: "offline-melody", BPM: bpm, Key: key.Root + modeFlag(key), Bars: 16},
		Theory:        schema.TheorySpec{Key: key.Root, Mode: key.Mode, Scale: key.Scale, OctaveRange: [2]int{4, 7}},
		StyleProfile:  profileName,
		Motif:         schema.MotifSpec{Length: 16, Steps: steps},
		Evolution:     buildEvolution(bpm, key),
		Automation:    schema.AutomationIntent{ModWheel: &schema.ModWheelIntent{Style: "subtle"}},
		VariationSeed: seed,
	}
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

func chooseBassProfile(key theory.Key, bpm float64, h []byte) string {
	candidates := []string{"bass_progressive"}
	if key.Mode == "minor" && bpm >= 126 {
		candidates = append(candidates, "bass_techno_driving")
	}
	if bpm < 118 {
		candidates = append(candidates, "bass_deep")
	}
	idx := int(h[0]) % len(candidates)
	return candidates[idx]
}

func chooseArpProfile(key theory.Key, bpm float64, h []byte) string {
	candidates := []string{"arp_flowing"}
	if bpm >= 128 {
		candidates = append(candidates, "arp_staccato")
	}
	if key.Mode == "minor" {
		candidates = append(candidates, "arp_dark")
	}
	idx := int(h[1]) % len(candidates)
	return candidates[idx]
}

func chooseMelodyProfile(key theory.Key, bpm float64, h []byte) string {
	candidates := []string{"melody_expressive"}
	if bpm >= 126 && key.Mode == "minor" {
		candidates = append(candidates, "melody_hypnotic")
	}
	if bpm < 110 {
		candidates = append(candidates, "melody_ambient")
	}
	idx := int(h[2]) % len(candidates)
	return candidates[idx]
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


