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
	steps := make([]StepPreview, 0, len(spec.Motif.Steps))
	for i, step := range spec.Motif.Steps {
		midi := 0
		if step.Active && step.Note != "" {
			if value, err := theory.NoteToMIDI(step.Note); err == nil {
				midi = value
			}
		}
		steps = append(steps, StepPreview{
			Step:     i,
			Note:     step.Note,
			MIDI:     midi,
			Active:   step.Active,
			Accent:   step.Accent,
			Ghost:    step.Ghost,
			Slide:    step.Slide,
			Legato:   step.Legato,
			Staccato: step.Staccato,
		})
	}
	return TrackPreview{
		PatternType: patternType,
		Label:       previewLabel(patternType),
		Steps:       steps,
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
