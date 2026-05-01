package schema

import "github.com/Andrea-Cavallo/cadenza/internal/theory"

// ArrangementScoreReport describes soft cross-track arrangement issues.
type ArrangementScoreReport struct {
	AllTracksPeakSameSection   bool
	PeakSectionByTrack         map[string]int
	MelodyArpPitchCollision    float64
	MelodyArpRegisterCollision float64
	Warnings                   []string
}

// ScoreArrangement checks whether bass, arp, and melody work together.
func ScoreArrangement(patterns map[string]*PatternSpec) ArrangementScoreReport {
	report := ArrangementScoreReport{PeakSectionByTrack: make(map[string]int)}
	for _, patternType := range []string{"bassline", "arpeggio", "melody"} {
		spec := patterns[patternType]
		if spec == nil {
			continue
		}
		densities := sectionDensities(spec.Motif.Steps, 4)
		report.PeakSectionByTrack[patternType] = peakSection(densities)
	}

	report.AllTracksPeakSameSection = allPeakSame(report.PeakSectionByTrack)
	report.MelodyArpPitchCollision = melodyArpPitchCollision(patterns, false)
	report.MelodyArpRegisterCollision = melodyArpPitchCollision(patterns, true)
	report.Warnings = arrangementWarnings(report)
	return report
}

func peakSection(densities []float64) int {
	if len(densities) == 0 {
		return 0
	}
	peakIdx := 0
	peakValue := densities[0]
	for i, density := range densities[1:] {
		if density > peakValue {
			peakIdx = i + 1
			peakValue = density
		}
	}
	return peakIdx + 1
}

func allPeakSame(peaks map[string]int) bool {
	if len(peaks) < 3 {
		return false
	}
	var first int
	for _, peak := range peaks {
		if first == 0 {
			first = peak
			continue
		}
		if peak != first {
			return false
		}
	}
	return true
}

func melodyArpPitchCollision(patterns map[string]*PatternSpec, includeRegister bool) float64 {
	arp := patterns["arpeggio"]
	melody := patterns["melody"]
	if arp == nil || melody == nil {
		return 0
	}
	limit := len(arp.Motif.Steps)
	if len(melody.Motif.Steps) < limit {
		limit = len(melody.Motif.Steps)
	}

	var simultaneous, collisions int
	for i := 0; i < limit; i++ {
		arpStep := arp.Motif.Steps[i]
		melodyStep := melody.Motif.Steps[i]
		if !arpStep.Active || !melodyStep.Active || arpStep.Note == "" || melodyStep.Note == "" {
			continue
		}
		simultaneous++
		if notesCollide(arpStep.Note, melodyStep.Note, includeRegister) {
			collisions++
		}
	}
	if simultaneous == 0 {
		return 0
	}
	return float64(collisions) / float64(simultaneous)
}

func notesCollide(a, b string, includeRegister bool) bool {
	if includeRegister {
		am, aerr := theory.NoteToMIDI(a)
		bm, berr := theory.NoteToMIDI(b)
		return aerr == nil && berr == nil && am == bm
	}
	return theory.NoteNameOnly(a) == theory.NoteNameOnly(b)
}

func arrangementWarnings(report ArrangementScoreReport) []string {
	var warnings []string
	if report.AllTracksPeakSameSection {
		warnings = append(warnings, "bassline, arpeggio, and melody all peak in the same section")
	}
	if report.MelodyArpPitchCollision > 0.5 {
		warnings = append(warnings, "melody and arpeggio collide on pitch class too often")
	}
	if report.MelodyArpRegisterCollision > 0.25 {
		warnings = append(warnings, "melody and arpeggio collide in the same register too often")
	}
	return warnings
}
