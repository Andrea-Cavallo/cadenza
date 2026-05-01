package generator

import (
	"fmt"
	"strings"
)

const (
	plannerVersion   = "planner-v1"
	styleCardVersion = "stylecards-v1"
)

// MusicalPlan captures the production intent that guides PatternSpec generation.
type MusicalPlan struct {
	PatternType        string
	StyleCard          StyleCard
	Mood               string
	ReferenceEnergy    string
	TensionCurve       string
	RhythmicIdentity   string
	MotifConcept       string
	DensityTarget      string
	CallResponsePlan   string
	SectionIntentions  []SectionIntention
	TrackSeparation    string
	RevisionPriorities []string
}

// StyleCard is a compact producer-facing direction for the LLM.
type StyleCard struct {
	Name        string
	Summary     string
	Groove      string
	Harmony     string
	Arrangement string
}

// SectionIntention describes what the pattern should do across a 4-bar section.
type SectionIntention struct {
	Bars    string
	Intent  string
	Density string
	Focus   string
}

// BuildMusicalPlan creates a deterministic plan from the musical context. The
// LLM still writes the notes, but it now receives a concrete production brief.
func BuildMusicalPlan(ctx MusicContext, patternType string) MusicalPlan {
	card := styleCardForContext(ctx)
	plan := MusicalPlan{
		PatternType:        patternType,
		StyleCard:          card,
		Mood:               moodForMode(ctx.Key.Mode),
		ReferenceEnergy:    energyForBPM(ctx.BPM),
		TensionCurve:       tensionCurveForStyle(card.Name),
		RhythmicIdentity:   rhythmicIdentity(patternType, card.Name),
		MotifConcept:       motifConcept(patternType, ctx.Key.Mode),
		DensityTarget:      densityTarget(patternType, card.Name),
		CallResponsePlan:   callResponsePlan(patternType),
		SectionIntentions:  sectionIntentions(patternType, card.Name),
		TrackSeparation:    trackSeparation(patternType),
		RevisionPriorities: revisionPriorities(patternType),
	}
	return plan
}

// PromptText renders the plan as a stable prompt block.
func (p MusicalPlan) PromptText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Style card: %s - %s\n", p.StyleCard.Name, p.StyleCard.Summary)
	fmt.Fprintf(&b, "Mood: %s\n", p.Mood)
	fmt.Fprintf(&b, "Reference energy: %s\n", p.ReferenceEnergy)
	fmt.Fprintf(&b, "Tension curve: %s\n", p.TensionCurve)
	fmt.Fprintf(&b, "Groove: %s\n", p.StyleCard.Groove)
	fmt.Fprintf(&b, "Harmony: %s\n", p.StyleCard.Harmony)
	fmt.Fprintf(&b, "Arrangement: %s\n", p.StyleCard.Arrangement)
	fmt.Fprintf(&b, "Rhythmic identity: %s\n", p.RhythmicIdentity)
	fmt.Fprintf(&b, "Motif concept: %s\n", p.MotifConcept)
	fmt.Fprintf(&b, "Density target: %s\n", p.DensityTarget)
	fmt.Fprintf(&b, "Call/response plan: %s\n", p.CallResponsePlan)
	fmt.Fprintf(&b, "Track separation: %s\n", p.TrackSeparation)
	b.WriteString("Section intentions:\n")
	for _, section := range p.SectionIntentions {
		fmt.Fprintf(&b, "- %s: %s; density %s; focus %s\n", section.Bars, section.Intent, section.Density, section.Focus)
	}
	b.WriteString("Revision priorities if the first draft is weak:\n")
	for _, priority := range p.RevisionPriorities {
		fmt.Fprintf(&b, "- %s\n", priority)
	}
	return strings.TrimSpace(b.String())
}

func styleCardForContext(ctx MusicContext) StyleCard {
	switch strings.ToLower(ctx.OfflineStyle) {
	case "hypnotic":
		return styleCards["minimal-hypnotic"]
	case "driving":
		return styleCards["warehouse-driving"]
	case "minimal":
		return styleCards["minimal-hypnotic"]
	case "melodic":
		return styleCards["festival-melodic"]
	}

	switch {
	case ctx.Key.Mode == "dorian":
		return styleCards["afterlife-dark"]
	case ctx.Key.Mode == "mixolydian" || ctx.Key.Mode == "lydian":
		return styleCards["festival-melodic"]
	case ctx.BPM >= 128:
		return styleCards["warehouse-driving"]
	default:
		return styleCards["prydz-progressive"]
	}
}

var styleCards = map[string]StyleCard{
	"afterlife-dark": {
		Name:        "afterlife-dark",
		Summary:     "dark melodic techno with patient tension, modal color, and cinematic restraint",
		Groove:      "steady four-on-floor pulse, syncopated off-beat details, no busy fills before the peak",
		Harmony:     "minor or Dorian chord tones first; color notes appear as controlled tension",
		Arrangement: "bass remains grounded, arp widens slowly, melody enters as a memorable restraint-heavy hook",
	},
	"prydz-progressive": {
		Name:        "prydz-progressive",
		Summary:     "wide progressive house phrasing with hypnotic repetition and clean emotional lift",
		Groove:      "locked grid, subtle syncopation, strong downbeat anchors, long-form build",
		Harmony:     "open fifths, chord thirds at emotional turns, clear resolution after tension",
		Arrangement: "bass drives the floor, arp provides motion, melody answers with simple high-register motifs",
	},
	"minimal-hypnotic": {
		Name:        "minimal-hypnotic",
		Summary:     "sparse afterhours loop design where silence and micro-variation create hypnosis",
		Groove:      "few hits, long gates, ghost notes only when they deepen the pocket",
		Harmony:     "root and fifth dominate; thirds and color tones appear rarely for slow evolution",
		Arrangement: "leave space between tracks; avoid simultaneous density peaks",
	},
	"festival-melodic": {
		Name:        "festival-melodic",
		Summary:     "bright melodic peak material with open voicings and a memorable emotional hook",
		Groove:      "confident momentum, accented lift into bars 9-12, clear payoff at the end",
		Harmony:     "major, Mixolydian, or Lydian color notes can define the peak",
		Arrangement: "arp carries width, melody owns the hook, bass stays simple and stable",
	},
	"warehouse-driving": {
		Name:        "warehouse-driving",
		Summary:     "dense peak-time techno pressure with tight rhythmic identity and controlled aggression",
		Groove:      "shorter gates, more active syncopation, firm accents, no wandering melodic clutter",
		Harmony:     "chord roots and fifths maintain drive; passing tones are brief and purposeful",
		Arrangement: "bass and arp may be busy, melody must stay concise to preserve impact",
	},
}

func moodForMode(mode string) string {
	switch mode {
	case "major":
		return "bright, open, anthemic"
	case "dorian":
		return "dark but lifted, underground and soulful"
	case "phrygian":
		return "tense, shadowy, dramatic"
	case "mixolydian":
		return "optimistic, festival-ready, spacious"
	case "lydian":
		return "floating, luminous, unresolved"
	default:
		return "dark, emotional, club-focused"
	}
}

func energyForBPM(bpm float64) string {
	switch {
	case bpm < 116:
		return "low-slung warmup energy"
	case bpm < 126:
		return "progressive mid-set energy"
	case bpm < 134:
		return "peak-time melodic drive"
	default:
		return "high-pressure techno energy"
	}
}

func tensionCurveForStyle(style string) string {
	switch style {
	case "minimal-hypnotic":
		return "slow arch: introduce space, add one detail, hold tension, resolve without overfilling"
	case "warehouse-driving":
		return "rising pressure: establish drive, thicken rhythm, peak in bars 9-12, tighten the ending"
	case "festival-melodic":
		return "emotional climb: introduce hook, widen harmony, peak high, resolve clearly"
	default:
		return "progressive build: motif first, rhythmic variation second, tension third, resolution or peak last"
	}
}

func rhythmicIdentity(patternType, style string) string {
	switch patternType {
	case "bassline":
		if style == "warehouse-driving" {
			return "root-driven pulse with syncopated pushes before chord changes"
		}
		return "hypnotic root anchor with sparse off-beat answers"
	case "arpeggio":
		if style == "minimal-hypnotic" {
			return "slow-evolving broken chord texture with breathing room"
		}
		return "open-voiced 16th-note motion with changing direction per section"
	default:
		return "short motif with rests, one memorable peak, and clear call-response shape"
	}
}

func motifConcept(patternType, mode string) string {
	switch patternType {
	case "bassline":
		return "one-bar groove cell anchored to each chord root, with fifth/color-tone answers"
	case "arpeggio":
		return "broken chord figure that changes inversion or direction every section"
	default:
		if mode == "lydian" || mode == "mixolydian" {
			return "small hook that climbs to the mode character note before resolving"
		}
		return "minimal hook that alternates chord tones with one controlled tension note"
	}
}

func densityTarget(patternType, style string) string {
	switch patternType {
	case "bassline":
		if style == "minimal-hypnotic" {
			return "8 active steps, no more unless needed for a pickup"
		}
		if style == "warehouse-driving" {
			return "11-13 active steps with purposeful syncopation"
		}
		return "9-11 active steps"
	case "arpeggio":
		if style == "minimal-hypnotic" {
			return "12 active steps with strategic rests"
		}
		return "14-16 active steps, varied articulation"
	default:
		if style == "festival-melodic" {
			return "7-10 active steps, hook-forward"
		}
		return "4-8 active steps, leave air around the phrase"
	}
}

func callResponsePlan(patternType string) string {
	switch patternType {
	case "bassline":
		return "root call on the downbeat, short answer before the next chord"
	case "arpeggio":
		return "low-to-high call answered by inversion or direction change"
	default:
		return "two-note call in the first half, answering phrase or peak in the second half"
	}
}

func sectionIntentions(patternType, style string) []SectionIntention {
	common := []SectionIntention{
		{Bars: "1-4", Intent: "introduce the core motif", Density: "low to medium", Focus: "identity and chord anchor"},
		{Bars: "5-8", Intent: "vary rhythm without losing recognizability", Density: "medium", Focus: "same motif, new placement"},
		{Bars: "9-12", Intent: "add tension or widen the phrase", Density: "highest", Focus: "peak color or denser motion"},
		{Bars: "13-16", Intent: "resolve, strip back, or deliver the final peak", Density: "controlled", Focus: "clear landing point"},
	}
	if patternType == "melody" && style == "minimal-hypnotic" {
		common[2].Density = "medium, not crowded"
		common[3].Intent = "resolve with one strong landing note"
	}
	return common
}

func trackSeparation(patternType string) string {
	switch patternType {
	case "bassline":
		return "stay low and rhythmically authoritative; do not compete with melody contour"
	case "arpeggio":
		return "own the mid/high texture but avoid becoming the main hook"
	default:
		return "own the emotional hook; stay sparse enough that the arpeggio remains audible"
	}
}

func revisionPriorities(patternType string) []string {
	base := []string{
		"remove identical 4-step phrases repeated more than twice",
		"strengthen chord tones on downbeats and section starts",
		"avoid flat articulation across long active runs",
	}
	if patternType == "melody" {
		return append(base, "make the contour memorable with one clear peak or landing note")
	}
	if patternType == "arpeggio" {
		return append(base, "vary arpeggio direction or inversion between sections")
	}
	return append(base, "keep the root/fifth anchor clear before adding passing notes")
}
