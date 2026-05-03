package main

import (
	"sort"

	"github.com/Andrea-Cavallo/cadenza/internal/generator"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

const previewStepsPerBar = 16

func buildGenerationPreview(musicCtx generator.MusicContext, specs map[string]*schema.PatternSpec) GenerationPreview {
	return GenerationPreview{
		Bars:        musicCtx.Bars,
		StepsPerBar: previewStepsPerBar,
		Chords:      buildChordPreview(musicCtx.ChordProgression),
		Patterns:    buildTrackPreviews(specs),
	}
}

func buildChordPreview(prog theory.ChordProgression) []ChordPreview {
	out := make([]ChordPreview, 0, len(prog.Chords))
	for _, chord := range prog.Chords {
		out = append(out, ChordPreview{
			Name:    chord.Root + theory.QualitySuffix(chord.Quality),
			FromBar: chord.Bars[0],
			ToBar:   chord.Bars[1],
		})
	}
	return out
}

func buildTrackPreviews(specs map[string]*schema.PatternSpec) []TrackPreview {
	order := []string{"bassline", "arpeggio", "melody"}
	out := make([]TrackPreview, 0, len(specs))
	for _, patternType := range order {
		spec := specs[patternType]
		if spec == nil {
			continue
		}
		out = append(out, buildTrackPreview(patternType, spec))
	}
	if len(out) > 0 {
		return out
	}

	keys := make([]string, 0, len(specs))
	for key := range specs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if specs[key] != nil {
			out = append(out, buildTrackPreview(key, specs[key]))
		}
	}
	return out
}

func buildTrackPreview(patternType string, spec *schema.PatternSpec) TrackPreview {
	motif := spec.Motif.Steps
	motifLen := len(motif)
	if motifLen == 0 {
		motifLen = 16
	}
	bars := spec.Meta.Bars
	if bars <= 0 {
		bars = 16
	}

	// Expand the motif across all bars so the piano roll shows the full arrangement.
	// Each bar repeats the same motif (evolution is applied by the renderer; here we
	// show the raw pattern repeated so the density and pitch content are clearly visible).
	totalSteps := bars * motifLen
	steps := make([]StepPreview, 0, totalSteps)

	for bar := 0; bar < bars; bar++ {
		for i, step := range motif {
			midi := 0
			if step.Active && step.Note != "" {
				if value, err := theory.NoteToMIDI(step.Note); err == nil {
					midi = value
				}
			}
			steps = append(steps, StepPreview{
				Step:          bar*motifLen + i,
				Note:          step.Note,
				MIDI:          midi,
				Active:        step.Active,
				Accent:        step.Accent,
				Ghost:         step.Ghost,
				Slide:         step.Slide,
				Legato:        step.Legato,
				Staccato:      step.Staccato,
				DurationSteps: previewDurationSteps(step),
				Velocity:      previewVelocity(step),
			})
		}
	}

	return TrackPreview{
		PatternType: patternType,
		Label:       previewLabel(patternType),
		Steps:       steps,
	}
}

func previewDurationSteps(step schema.StepSpec) int {
	switch {
	case step.Slide || step.Legato:
		return 2
	case step.Staccato:
		return 1
	default:
		return 1
	}
}

func previewVelocity(step schema.StepSpec) int {
	switch {
	case step.Ghost:
		return 46
	case step.Accent:
		return 110
	default:
		return 84
	}
}

func previewLabel(patternType string) string {
	switch patternType {
	case "bassline":
		return "Bass"
	case "arpeggio":
		return "Arp"
	case "melody":
		return "Melody"
	default:
		return patternType
	}
}
