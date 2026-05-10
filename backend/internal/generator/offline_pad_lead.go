package generator

import (
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// chordPadTemplate generates a sustain chord voicing layer (CH6).
// Drop-2 voicing: root low, fifth mid, third high.
// Density and octave adapt to StyleFamily.
func chordPadTemplate(ctx MusicContext) *schema.PatternSpec {
	key := ctx.Key
	prog := ctx.ChordProgression
	seed := ctx.VariationSeed
	bpm := ctx.BPM
	sf := ctx.StyleFamily

	h := seedHash(seed + key.Root + key.Scale + "chord_pad")
	steps := make([]schema.StepSpec, motifSteps)

	for i, chord := range prog.Chords {
		base := i * stepsPerSection
		fillChordPadSection(steps, base, chord, sf, h, i)
	}

	ensureGateVariation(steps)
	profileName := chordPadProfile(sf)

	sweepStyle := "subtle"
	if sf == StyleFamilySub {
		sweepStyle = "medium"
	}

	return &schema.PatternSpec{
		SpecVersion:   "1.0",
		PatternType:   "chord_pad",
		Meta:          schema.PatternMeta{Name: "offline-chord-pad", BPM: bpm, Key: key.Root + modeFlag(key), Bars: 4},
		Theory:        schema.TheorySpec{Key: key.Root, Mode: key.Mode, Scale: key.Scale, OctaveRange: [2]int{3, 5}},
		StyleProfile:  profileName,
		Motif:         schema.MotifSpec{Length: motifSteps, Steps: steps},
		Evolution:     buildPadEvolution(bpm),
		Automation:    schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: sweepStyle}},
		VariationSeed: seed,
	}
}

// fillChordPadSection writes one 16-step chord pad section using drop-2 voicing.
// Groove:   4 downbeats — root3 + fifth4 + third5, legato gate 90%.
// Rolling:  2 hits (beat 1 and 9) — root3 + fifth4 only, minimal.
// Sub:      6 hits + lower octave voicing — root2 + fifth3 + third4.
func fillChordPadSection(steps []schema.StepSpec, base int, chord theory.ProgressionChord, sf StyleFamily, h []byte, sectionIdx int) {
	notes, err := theory.ChordNotes(chord.Root, chord.Quality)
	if err != nil || len(notes) < 3 {
		notes = []string{chord.Root, chord.Root, chord.Root}
	}

	padNote := func(name, oct string) string {
		full := name + oct
		midi, mErr := theory.NoteToMIDI(full)
		if mErr != nil {
			return full
		}
		if midi < 48 {
			return name + "4"
		}
		if midi > 79 {
			return name + "4"
		}
		return full
	}

	switch sf {
	case StyleFamilyRolling:
		// Sparse: only beat 0 and beat 8 (steps 0 and 8 within section).
		// Two-voice drop (no third) for maximum space.
		steps[base+0] = schema.StepSpec{Active: true, Note: padNote(notes[0], "3"), Accent: true, Legato: true}
		steps[base+8] = schema.StepSpec{Active: true, Note: padNote(notes[2], "4"), Legato: true}

	case StyleFamilySub:
		// Lower octave voicing, 6 hits across the section.
		steps[base+0] = schema.StepSpec{Active: true, Note: padNote(notes[0], "2"), Accent: true, Legato: true}
		steps[base+4] = schema.StepSpec{Active: true, Note: padNote(notes[2], "3"), Legato: true}
		steps[base+6] = schema.StepSpec{Active: true, Note: padNote(notes[1], "3"), Ghost: true}
		steps[base+8] = schema.StepSpec{Active: true, Note: padNote(notes[0], "2"), Legato: true}
		steps[base+12] = schema.StepSpec{Active: true, Note: padNote(notes[2], "3"), Legato: true}
		// Tension chord: add seventh on section 2 (h-gated)
		if sectionIdx == 2 && h[(base+14)%32]%2 == 0 {
			steps[base+14] = schema.StepSpec{Active: true, Note: padNote(notes[0], "3"), Ghost: true}
		}

	default: // Groove: 4 downbeats, full drop-2 voicing.
		steps[base+0] = schema.StepSpec{Active: true, Note: padNote(notes[0], "3"), Accent: true, Legato: true}
		steps[base+4] = schema.StepSpec{Active: true, Note: padNote(notes[2], "4"), Legato: true}
		steps[base+8] = schema.StepSpec{Active: true, Note: padNote(notes[1], "5"), Legato: true}
		steps[base+12] = schema.StepSpec{Active: true, Note: padNote(notes[0], "3"), Ghost: true}
	}
}

func chordPadProfile(sf StyleFamily) string {
	switch sf {
	case StyleFamilyRolling:
		return "chord_pad_rolling"
	case StyleFamilySub:
		return "chord_pad_sub"
	default:
		return "chord_pad_groove"
	}
}

func buildPadEvolution(bpm float64) []schema.EvolutionStep {
	return []schema.EvolutionStep{
		{FromBar: 1, ToBar: 1, Action: "introduce", Intensity: 0.4},
		{FromBar: 2, ToBar: 2, Action: "build", Intensity: 0.6},
		{FromBar: 3, ToBar: 3, Action: "peak", Intensity: 0.8},
		{FromBar: 4, ToBar: 4, Action: "release", Intensity: 0.4},
	}
}

// leadPluckTemplate generates a staccato lead layer (CH7) — same contour as
// melody but with all articulations overridden to percussive staccato.
func leadPluckTemplate(ctx MusicContext) *schema.PatternSpec {
	key := ctx.Key
	prog := ctx.ChordProgression
	seed := ctx.VariationSeed
	bpm := ctx.BPM
	sf := ctx.StyleFamily

	// Build melody contour first, then override articulation.
	melCtx := ctx
	melCtx.VariationSeed = seed + "_lead" // different seed so lead ≠ melody
	melSpec := melodyTemplate(melCtx)
	if melSpec == nil {
		return nil
	}

	steps := make([]schema.StepSpec, len(melSpec.Motif.Steps))
	copy(steps, melSpec.Motif.Steps)

	// Override: every active note becomes staccato, no legato, no ghost.
	for i := range steps {
		if !steps[i].Active {
			continue
		}
		steps[i].Ghost = false
		steps[i].Legato = false
		steps[i].Staccato = true
	}

	// StyleFamily adaptation: thin density for sub, boost for rolling.
	steps = applyLeadDensity(steps, prog, sf)

	return &schema.PatternSpec{
		SpecVersion:   "1.0",
		PatternType:   "lead_stab",
		Meta:          schema.PatternMeta{Name: "offline-lead", BPM: bpm, Key: key.Root + modeFlag(key), Bars: 4},
		Theory:        schema.TheorySpec{Key: key.Root, Mode: key.Mode, Scale: key.Scale, OctaveRange: [2]int{4, 6}},
		StyleProfile:  "lead_stab",
		Motif:         schema.MotifSpec{Length: len(steps), Steps: steps},
		Evolution:     buildLeadEvolution(bpm, sf),
		Automation:    schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: "subtle"}},
		VariationSeed: seed,
	}
}

// applyLeadDensity adapts active step count based on StyleFamily.
// Rolling: keep all, Sub: thin to 1-3 per section, Groove: keep all.
func applyLeadDensity(steps []schema.StepSpec, prog theory.ChordProgression, sf StyleFamily) []schema.StepSpec {
	if sf != StyleFamilySub {
		return steps
	}
	// Sub: deactivate all but the accent downbeat per section.
	for i := range prog.Chords {
		base := i * stepsPerSection
		sl := steps[base : base+stepsPerSection]
		kept := false
		for j := range sl {
			if sl[j].Active && sl[j].Accent && !kept {
				kept = true
				continue
			}
			if sl[j].Active && !kept {
				kept = true
				continue
			}
			sl[j].Active = false
		}
	}
	return steps
}

func buildLeadEvolution(bpm float64, sf StyleFamily) []schema.EvolutionStep {
	switch sf {
	case StyleFamilyRolling:
		return []schema.EvolutionStep{
			{FromBar: 1, ToBar: 1, Action: "build", Intensity: 0.6},
			{FromBar: 2, ToBar: 2, Action: "peak", Intensity: 0.9},
			{FromBar: 3, ToBar: 3, Action: "peak", Intensity: 1.0},
			{FromBar: 4, ToBar: 4, Action: "release", Intensity: 0.5},
		}
	case StyleFamilySub:
		return []schema.EvolutionStep{
			{FromBar: 1, ToBar: 1, Action: "introduce", Intensity: 0.2},
			{FromBar: 2, ToBar: 2, Action: "build", Intensity: 0.3},
			{FromBar: 3, ToBar: 3, Action: "peak", Intensity: 0.4},
			{FromBar: 4, ToBar: 4, Action: "release", Intensity: 0.2},
		}
	default:
		return []schema.EvolutionStep{
			{FromBar: 1, ToBar: 1, Action: "introduce", Intensity: 0.4},
			{FromBar: 2, ToBar: 2, Action: "build", Intensity: 0.6},
			{FromBar: 3, ToBar: 3, Action: "peak", Intensity: 0.8},
			{FromBar: 4, ToBar: 4, Action: "release", Intensity: 0.4},
		}
	}
}
