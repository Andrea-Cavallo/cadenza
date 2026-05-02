package schema

import (
	"math"
	"strings"

	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

// MusicalScoreReport contains soft musical-quality metrics. These do not
// replace validation; they explain whether a valid pattern is musically strong.
type MusicalScoreReport struct {
	RepeatedPhraseCount    int
	DownbeatChordToneRatio float64
	ChordToneStrength      float64
	MotifClarity           float64
	PitchContourDiversity  float64
	VelocityFlatness       float64
	DensityBalance         float64
	TensionArcScore        float64
	TrackSeparationScore   float64
	SectionDensities       []float64
	Warnings               []string
}

// ScoreMusicality evaluates soft musical quality against a chord progression.
func (v *Validator) ScoreMusicality(spec *PatternSpec, prog theory.ChordProgression) MusicalScoreReport {
	report := MusicalScoreReport{
		RepeatedPhraseCount:    repeatedPhraseCount(spec.Motif.Steps),
		DownbeatChordToneRatio: downbeatChordToneRatio(spec.Motif.Steps, prog),
		ChordToneStrength:      chordToneStrength(spec.Motif.Steps, prog),
		PitchContourDiversity:  pitchContourDiversity(spec.Motif.Steps),
		VelocityFlatness:       articulationFlatness(spec.Motif.Steps),
		SectionDensities:       sectionDensities(spec.Motif.Steps, len(prog.Chords)),
	}
	report.MotifClarity = motifClarity(spec.Motif.Steps)
	report.DensityBalance = densityBalance(report.SectionDensities)
	report.TensionArcScore = tensionArcScore(report.SectionDensities)
	report.TrackSeparationScore = defaultTrackSeparationScore(spec.PatternType, report)
	report.Warnings = musicalWarnings(spec.PatternType, report)
	return report
}

func repeatedPhraseCount(steps []StepSpec) int {
	const phraseLen = 4
	seen := make(map[string]int)
	for i := 0; i+phraseLen <= len(steps); i += phraseLen {
		seen[phraseFingerprint(steps[i:i+phraseLen])]++
	}

	repeated := 0
	for _, count := range seen {
		if count > 2 {
			repeated += count - 2
		}
	}
	return repeated
}

func phraseFingerprint(steps []StepSpec) string {
	var b strings.Builder
	for _, step := range steps {
		switch {
		case !step.Active:
			b.WriteByte('0')
		case step.Accent:
			b.WriteByte('A')
		case step.Ghost:
			b.WriteByte('G')
		case step.Slide:
			b.WriteByte('S')
		case step.Legato:
			b.WriteByte('L')
		case step.Staccato:
			b.WriteByte('T')
		default:
			b.WriteByte('1')
		}
		if step.Note != "" {
			b.WriteString(theory.NoteNameOnly(step.Note))
		}
	}
	return b.String()
}

func downbeatChordToneRatio(steps []StepSpec, prog theory.ChordProgression) float64 {
	if len(prog.Chords) == 0 || len(steps) == 0 {
		return 0
	}
	stepsPerSection := len(steps) / len(prog.Chords)
	if stepsPerSection == 0 {
		return 0
	}

	var activeDownbeats, chordToneHits int
	for i, chord := range prog.Chords {
		idx := i * stepsPerSection
		if idx >= len(steps) || !steps[idx].Active || steps[idx].Note == "" {
			continue
		}
		activeDownbeats++
		chordTones, err := theory.ChordNotes(chord.Root, chord.Quality)
		if err == nil && isChordTone(theory.NoteNameOnly(steps[idx].Note), chordTones) {
			chordToneHits++
		}
	}
	if activeDownbeats == 0 {
		return 0
	}
	return float64(chordToneHits) / float64(activeDownbeats)
}

func pitchContourDiversity(steps []StepSpec) float64 {
	var last *int
	directions := make(map[int]bool)
	for _, step := range steps {
		if !step.Active || step.Note == "" {
			continue
		}
		midi, err := theory.NoteToMIDI(step.Note)
		if err != nil {
			continue
		}
		if last != nil {
			delta := midi - *last
			switch {
			case delta > 0:
				directions[1] = true
			case delta < 0:
				directions[-1] = true
			default:
				directions[0] = true
			}
		}
		last = &midi
	}
	return float64(len(directions)) / 3.0
}

func chordToneStrength(steps []StepSpec, prog theory.ChordProgression) float64 {
	if len(prog.Chords) == 0 || len(steps) == 0 {
		return 0
	}
	stepsPerSection := len(steps) / len(prog.Chords)
	if stepsPerSection == 0 {
		return 0
	}
	var active, hits int
	for i, chord := range prog.Chords {
		start := i * stepsPerSection
		end := start + stepsPerSection
		if i == len(prog.Chords)-1 {
			end = len(steps)
		}
		chordTones, err := theory.ChordNotes(chord.Root, chord.Quality)
		if err != nil {
			continue
		}
		a, h := chordToneStats(steps[start:end], chordTones)
		active += a
		hits += h
	}
	if active == 0 {
		return 0
	}
	return float64(hits) / float64(active)
}

func motifClarity(steps []StepSpec) float64 {
	active := 0
	pitches := make(map[string]bool)
	for _, step := range steps {
		if !step.Active || step.Note == "" {
			continue
		}
		active++
		pitches[theory.NoteNameOnly(step.Note)] = true
	}
	if active == 0 {
		return 0
	}
	if len(pitches) <= 4 {
		return 1
	}
	return math.Max(0, 1-float64(len(pitches)-4)/float64(active))
}

func articulationFlatness(steps []StepSpec) float64 {
	longestRun, currentRun := 0, 0
	last := ""
	for _, step := range steps {
		if !step.Active {
			currentRun = 0
			last = ""
			continue
		}
		current := articulationClass(step)
		if current == last {
			currentRun++
		} else {
			currentRun = 1
			last = current
		}
		if currentRun > longestRun {
			longestRun = currentRun
		}
	}
	if len(steps) == 0 {
		return 0
	}
	return math.Min(1, float64(longestRun)/float64(len(steps)))
}

func articulationClass(step StepSpec) string {
	switch {
	case step.Ghost:
		return "ghost"
	case step.Accent:
		return "accent"
	case step.Slide:
		return "slide"
	case step.Legato:
		return "legato"
	case step.Staccato:
		return "staccato"
	default:
		return "plain"
	}
}

func sectionDensities(steps []StepSpec, sections int) []float64 {
	if sections <= 0 {
		sections = 4
	}
	stepsPerSection := len(steps) / sections
	if stepsPerSection == 0 {
		return nil
	}

	out := make([]float64, 0, sections)
	for i := 0; i < sections; i++ {
		start := i * stepsPerSection
		end := start + stepsPerSection
		if i == sections-1 {
			end = len(steps)
		}
		active := 0
		for _, step := range steps[start:end] {
			if step.Active {
				active++
			}
		}
		out = append(out, float64(active)/float64(end-start))
	}
	return out
}

func densityBalance(densities []float64) float64 {
	if len(densities) == 0 {
		return 0
	}
	minDensity, maxDensity := densities[0], densities[0]
	for _, density := range densities[1:] {
		if density < minDensity {
			minDensity = density
		}
		if density > maxDensity {
			maxDensity = density
		}
	}
	if maxDensity == 0 {
		return 0
	}
	return 1 - math.Min(1, maxDensity-minDensity)
}

func tensionArcScore(densities []float64) float64 {
	if len(densities) < 4 {
		return 0
	}
	score := 0.5
	if densities[2] >= densities[0] {
		score += 0.25
	}
	if densities[3] <= densities[2] || densities[3] >= densities[1] {
		score += 0.25
	}
	return math.Min(1, score)
}

func defaultTrackSeparationScore(patternType string, report MusicalScoreReport) float64 {
	switch patternType {
	case "bassline":
		return clampScore(1 - report.PitchContourDiversity*0.25)
	case "arpeggio":
		return clampScore(0.75 + report.PitchContourDiversity*0.25)
	default:
		return clampScore(1 - report.VelocityFlatness)
	}
}

func musicalWarnings(patternType string, report MusicalScoreReport) []string {
	var warnings []string
	if report.RepeatedPhraseCount > 0 {
		warnings = append(warnings, "4-step phrase repeats more than twice")
	}
	if report.DownbeatChordToneRatio < 0.75 && patternType != "melody" {
		warnings = append(warnings, "downbeats need stronger chord-tone anchoring")
	}
	if report.PitchContourDiversity < 0.34 && patternType != "bassline" {
		warnings = append(warnings, "pitch contour is too static")
	}
	if report.VelocityFlatness > 0.35 {
		warnings = append(warnings, "articulation is too flat across active runs")
	}
	if report.TensionArcScore < 0.75 {
		warnings = append(warnings, "section tension arc does not clearly build and resolve")
	}
	return warnings
}

func clampScore(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
