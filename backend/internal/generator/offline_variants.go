package generator

import (
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// basslineRolling generates an acid/TB-303 style bassline with near-full density.
// All 4 steps in each sub-group are active; groove comes from velocity variation.
func basslineRolling(ctx MusicContext) *schema.PatternSpec {
	key := ctx.Key
	prog := ctx.ChordProgression
	seed := ctx.VariationSeed
	bpm := ctx.BPM

	h := seedHash(seed + key.Root + key.Scale + "bass_rolling")
	palette := chooseRollingPalette(bpm, key)

	steps := make([]schema.StepSpec, motifSteps)

	for i, chord := range prog.Chords {
		base := i * stepsPerSection
		root := chord.Root + "2"
		fifth := chordFifth(chord) + "2"
		color := bassColorTone(key, chord) + "2"
		approach := approachNote(chord.Root, key) + "2"

		highRoot := chord.Root + "3"
		if m, _ := theory.NoteToMIDI(highRoot); m > 55 {
			highRoot = root
		}

		pi := int(h[(i*7+2)%32]) % len(palette)
		p0 := palette[pi]
		p1 := palette[(pi+1+int(h[(i*7+4)%32]))%len(palette)]

		// A-B-A-B groove: beats 1+3 share base pattern, beats 2+4 share variation.
		// Root↔fifth swap on echo positions preserves anti-loop variance.
		fillBassSubPattern(steps, base+0, p0, root, fifth, color, highRoot, approach, key)
		fillBassSubPattern(steps, base+4, p1, root, fifth, color, highRoot, approach, key)
		fillBassSubPattern(steps, base+8, p0, fifth, root, color, highRoot, approach, key)
		fillBassSubPattern(steps, base+12, p1, fifth, root, color, highRoot, approach, key)
	}

	// Rolling: no style thinning — density target is 14-16 per section.
	ensureBassMinDensityBounded(steps, prog, 14, "2")
	ensureGateVariation(steps)

	return &schema.PatternSpec{
		SpecVersion:   "1.0",
		PatternType:   "bassline",
		Meta:          schema.PatternMeta{Name: "offline-bass-rolling", BPM: bpm, Key: key.Root + modeFlag(key), Bars: 4},
		Theory:        schema.TheorySpec{Key: key.Root, Mode: key.Mode, Scale: key.Scale, OctaveRange: [2]int{1, 3}},
		StyleProfile:  "bass_rolling",
		Motif:         schema.MotifSpec{Length: motifSteps, Steps: steps},
		Evolution:     buildBassEvolution(bpm, key),
		Automation:    schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: "dramatic"}},
		VariationSeed: seed,
	}
}

// basslineSub generates a deep/dub-techno bassline with minimal density (4-6/16).
// Notes live in octave 1 (MIDI 24-43) for true sub-bass territory.
func basslineSub(ctx MusicContext) *schema.PatternSpec {
	key := ctx.Key
	prog := ctx.ChordProgression
	seed := ctx.VariationSeed
	bpm := ctx.BPM

	h := seedHash(seed + key.Root + key.Scale + "bass_sub")
	palette := chooseSubPalette(bpm, key)

	steps := make([]schema.StepSpec, motifSteps)

	for i, chord := range prog.Chords {
		base := i * stepsPerSection
		// Octave 1 for true sub-bass; highRoot bumps to octave 2.
		root := chord.Root + "1"
		fifth := chordFifth(chord) + "1"
		color := bassColorTone(key, chord) + "1"
		approach := approachNote(chord.Root, key) + "1"

		highRoot := chord.Root + "2"
		if m, _ := theory.NoteToMIDI(highRoot); m > 43 {
			highRoot = root
		}

		pi := int(h[(i*7+2)%32]) % len(palette)
		p0 := palette[pi]
		p1 := palette[(pi+1+int(h[(i*7+4)%32]))%len(palette)]

		fillBassSubPattern(steps, base+0, p0, root, fifth, color, highRoot, approach, key)
		fillBassSubPattern(steps, base+4, p1, root, fifth, color, highRoot, approach, key)
		fillBassSubPattern(steps, base+8, p0, fifth, root, color, highRoot, approach, key)
		fillBassSubPattern(steps, base+12, p1, fifth, root, color, highRoot, approach, key)
	}

	// Sub: keep silence intact — only enforce minimum 4 active per section.
	ensureBassMinDensityBounded(steps, prog, 4, "1")
	ensureBassMaxDensityBounded(steps, 6)
	ensureGateVariation(steps)

	return &schema.PatternSpec{
		SpecVersion:   "1.0",
		PatternType:   "bassline",
		Meta:          schema.PatternMeta{Name: "offline-bass-sub", BPM: bpm, Key: key.Root + modeFlag(key), Bars: 4},
		Theory:        schema.TheorySpec{Key: key.Root, Mode: key.Mode, Scale: key.Scale, OctaveRange: [2]int{1, 2}},
		StyleProfile:  "bass_sub",
		Motif:         schema.MotifSpec{Length: motifSteps, Steps: steps},
		Evolution:     buildBassEvolution(bpm, key),
		Automation:    schema.AutomationIntent{FilterSweep: &schema.FilterSweepIntent{Style: "medium"}},
		VariationSeed: seed,
	}
}
