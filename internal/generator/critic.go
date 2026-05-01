package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Andrea-Cavallo/cadenza/internal/llm"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
)

const (
	criticVersion         = "critic-v1"
	revisionPolicyVersion = "revision-policy-v1"
)

// CriticReport is the LLM-readable and machine-readable quality assessment for
// a valid PatternSpec. It is intentionally separate from hard validation.
type CriticReport struct {
	PatternType             string   `json:"pattern_type"`
	NeedsRevision           bool     `json:"needs_revision"`
	RepetitionScore         float64  `json:"repetition_score"`
	MotifClarityScore       float64  `json:"motif_clarity_score"`
	ChordToneStrengthScore  float64  `json:"chord_tone_strength_score"`
	ContourScore            float64  `json:"contour_score"`
	DensityScore            float64  `json:"density_score"`
	TensionArcScore         float64  `json:"tension_arc_score"`
	TrackSeparationScore    float64  `json:"track_separation_score"`
	Problems                []string `json:"problems"`
	RevisionInstructions    []string `json:"revision_instructions"`
	KeepElements            []string `json:"keep_elements"`
	RevisionPromptVersion   string   `json:"revision_prompt_version"`
	OneRevisionRoundApplied bool     `json:"one_revision_round_applied"`
}

func (g *SingleGenerator) critiqueAndMaybeRevise(
	ctx context.Context,
	req llm.GenerateRequest,
	rawJSON []byte,
	spec *schema.PatternSpec,
	musicCtx MusicContext,
	plan MusicalPlan,
) (finalSpec *schema.PatternSpec, finalJSON []byte) {
	score := g.validator.ScoreMusicality(spec, musicCtx.ChordProgression)
	report := g.localCriticReport(spec, plan, score)

	llmReport, err := g.runCritic(ctx, req, rawJSON, plan, score)
	if err != nil {
		slog.Warn("critic failed, using local musical score",
			"type", spec.PatternType,
			"seed", musicCtx.VariationSeed,
			"error", err,
		)
	} else {
		report = mergeCriticReports(report, llmReport)
	}

	slog.Info("critic report",
		"type", spec.PatternType,
		"seed", musicCtx.VariationSeed,
		"needs_revision", report.NeedsRevision,
		"problems", report.Problems,
		"revision_instructions", report.RevisionInstructions,
	)

	if !report.NeedsRevision {
		return spec, rawJSON
	}

	revised, revisedRaw, err := g.runSingleRevision(ctx, req, rawJSON, report)
	if err != nil {
		slog.Warn("targeted revision failed, keeping original valid PatternSpec",
			"type", spec.PatternType,
			"seed", musicCtx.VariationSeed,
			"error", err,
		)
		return spec, rawJSON
	}

	if err := g.validator.ValidateWithChords(revised, musicCtx.ChordProgression); err != nil {
		slog.Warn("targeted revision did not validate, keeping original valid PatternSpec",
			"type", spec.PatternType,
			"seed", musicCtx.VariationSeed,
			"error", err,
		)
		return spec, rawJSON
	}

	revisedScore := g.validator.ScoreMusicality(revised, musicCtx.ChordProgression)
	slog.Info("targeted revision accepted",
		"type", spec.PatternType,
		"seed", musicCtx.VariationSeed,
		"repeated_phrases", revisedScore.RepeatedPhraseCount,
		"downbeat_chord_tone_ratio", revisedScore.DownbeatChordToneRatio,
		"pitch_contour_diversity", revisedScore.PitchContourDiversity,
		"velocity_flatness", revisedScore.VelocityFlatness,
		"warnings", revisedScore.Warnings,
	)
	return revised, revisedRaw
}

func (g *SingleGenerator) runCritic(
	ctx context.Context,
	baseReq llm.GenerateRequest,
	rawJSON []byte,
	plan MusicalPlan,
	score schema.MusicalScoreReport,
) (CriticReport, error) {
	req := llm.GenerateRequest{
		System:       criticSystemPrompt,
		Messages:     []llm.Message{{Role: "user", Content: criticPrompt(rawJSON, plan, score)}},
		OutputSchema: []byte(exampleCriticJSON()),
		SchemaProperties: map[string]any{
			"pattern_type":               map[string]any{"type": "string"},
			"needs_revision":             map[string]any{"type": "boolean"},
			"repetition_score":           map[string]any{"type": "number"},
			"motif_clarity_score":        map[string]any{"type": "number"},
			"chord_tone_strength_score":  map[string]any{"type": "number"},
			"contour_score":              map[string]any{"type": "number"},
			"density_score":              map[string]any{"type": "number"},
			"tension_arc_score":          map[string]any{"type": "number"},
			"track_separation_score":     map[string]any{"type": "number"},
			"problems":                   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"revision_instructions":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"keep_elements":              map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"revision_prompt_version":    map[string]any{"type": "string"},
			"one_revision_round_applied": map[string]any{"type": "boolean"},
		},
		SchemaRequired: []string{
			"pattern_type", "needs_revision", "repetition_score", "motif_clarity_score",
			"chord_tone_strength_score", "contour_score", "density_score", "tension_arc_score",
			"track_separation_score", "problems", "revision_instructions", "keep_elements",
			"revision_prompt_version", "one_revision_round_applied",
		},
		Temperature: baseReq.Temperature,
		MaxTokens:   1200,
	}

	resp, err := g.provider.Generate(ctx, req)
	if err != nil {
		return CriticReport{}, err
	}
	var report CriticReport
	if err := json.Unmarshal(resp.RawJSON, &report); err != nil {
		return CriticReport{}, fmt.Errorf("critic parse: %w", err)
	}
	return report, nil
}

func (g *SingleGenerator) runSingleRevision(
	ctx context.Context,
	baseReq llm.GenerateRequest,
	rawJSON []byte,
	report CriticReport,
) (*schema.PatternSpec, []byte, error) {
	req := baseReq
	req.Messages = append(req.Messages, llm.Message{Role: "assistant", Content: string(rawJSON)})
	req.Messages = append(req.Messages, llm.Message{Role: "user", Content: revisionPrompt(report)})
	req.MaxTokens = baseReq.MaxTokens

	resp, err := g.provider.Generate(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	var revised schema.PatternSpec
	if err := json.Unmarshal(resp.RawJSON, &revised); err != nil {
		return nil, nil, fmt.Errorf("revision parse: %w", err)
	}
	return &revised, resp.RawJSON, nil
}

func (g *SingleGenerator) localCriticReport(spec *schema.PatternSpec, plan MusicalPlan, score schema.MusicalScoreReport) CriticReport {
	problems := append([]string{}, score.Warnings...)
	problems = append(problems, planAlignmentWarnings(spec, plan, score)...)
	instructions := targetedInstructions(spec.PatternType, problems)

	return CriticReport{
		PatternType:             spec.PatternType,
		NeedsRevision:           len(problems) > 0,
		RepetitionScore:         clamp01(1 - float64(score.RepeatedPhraseCount)*0.35),
		MotifClarityScore:       clamp01(score.MotifClarity),
		ChordToneStrengthScore:  clamp01(score.ChordToneStrength),
		ContourScore:            clamp01(score.PitchContourDiversity),
		DensityScore:            clamp01(score.DensityBalance),
		TensionArcScore:         clamp01(score.TensionArcScore),
		TrackSeparationScore:    clamp01(score.TrackSeparationScore),
		Problems:                problems,
		RevisionInstructions:    instructions,
		KeepElements:            []string{"keep valid scale notes", "keep chord progression alignment", "keep the current style_profile unless it is invalid"},
		RevisionPromptVersion:   revisionPolicyVersion,
		OneRevisionRoundApplied: false,
	}
}

func mergeCriticReports(local, llmReport CriticReport) CriticReport {
	if llmReport.PatternType == "" {
		return local
	}
	if len(llmReport.Problems) == 0 && len(local.Problems) > 0 {
		llmReport.Problems = local.Problems
	}
	if len(llmReport.RevisionInstructions) == 0 && len(local.RevisionInstructions) > 0 {
		llmReport.RevisionInstructions = local.RevisionInstructions
	}
	if len(llmReport.KeepElements) == 0 {
		llmReport.KeepElements = local.KeepElements
	}
	llmReport.NeedsRevision = llmReport.NeedsRevision || local.NeedsRevision
	if llmReport.RevisionPromptVersion == "" {
		llmReport.RevisionPromptVersion = revisionPolicyVersion
	}
	return llmReport
}

const criticSystemPrompt = `You are a strict electronic music production critic.
Evaluate an already-valid PatternSpec for musical quality only.
Return ONLY valid JSON matching the critic report schema.
Do not rewrite the PatternSpec in this step.`

func criticPrompt(rawJSON []byte, plan MusicalPlan, score schema.MusicalScoreReport) string {
	scoreJSON, _ := json.MarshalIndent(score, "", "  ")
	return fmt.Sprintf(`<musical_plan>
%s
</musical_plan>

<go_musical_score>
%s
</go_musical_score>

<pattern_spec>
%s
</pattern_spec>

Score repetition, motif clarity, chord-tone strength, contour, density, tension arc, and track separation from 0.0 to 1.0.
Set needs_revision=true only when a targeted one-round correction would clearly improve the part.
Give precise revision_instructions that fix only weak dimensions and preserve strong material.`, plan.PromptText(), string(scoreJSON), string(rawJSON))
}

func revisionPrompt(report CriticReport) string {
	return fmt.Sprintf(`<critic_report>
%s
</critic_report>

Revise the previous PatternSpec once.
Fix only the listed problems.
Preserve the JSON schema, variation_seed, key, scale, pattern_type, and chord progression alignment.
%s
Keep these elements when possible: %s
Return ONLY the revised PatternSpec JSON.`, mustJSON(report), densityConstraintNote(report.PatternType), strings.Join(report.KeepElements, "; "))
}

func densityConstraintNote(patternType string) string {
	switch patternType {
	case "bassline":
		return "CRITICAL: keep the total number of active steps between 8 and 13. Do NOT remove notes below 8 active steps."
	case "arpeggio":
		return "CRITICAL: keep the total number of active steps between 12 and 16. Do NOT remove notes below 12 active steps."
	case "melody":
		return "CRITICAL: keep the total number of active steps between 4 and 10."
	default:
		return ""
	}
}

func exampleCriticJSON() string {
	return mustJSON(CriticReport{
		PatternType:             "bassline",
		NeedsRevision:           true,
		RepetitionScore:         0.7,
		MotifClarityScore:       0.8,
		ChordToneStrengthScore:  0.9,
		ContourScore:            0.5,
		DensityScore:            0.8,
		TensionArcScore:         0.6,
		TrackSeparationScore:    0.8,
		Problems:                []string{"4-step phrase repeats more than twice"},
		RevisionInstructions:    []string{"change the bars 9-12 rhythm while preserving chord tones"},
		KeepElements:            []string{"keep the downbeat chord tones"},
		RevisionPromptVersion:   revisionPolicyVersion,
		OneRevisionRoundApplied: false,
	})
}

func mustJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func targetedInstructions(patternType string, problems []string) []string {
	if len(problems) == 0 {
		return nil
	}
	instructions := make([]string, 0, len(problems))
	for _, problem := range problems {
		switch {
		case strings.Contains(problem, "repeats"):
			instructions = append(instructions, "alter one repeated 4-step phrase using rhythmic placement, pitch, or articulation while preserving the motif identity")
		case strings.Contains(problem, "downbeat") || strings.Contains(problem, "chord"):
			instructions = append(instructions, "move section downbeats and strong accents onto active chord tones")
		case strings.Contains(problem, "contour"):
			instructions = append(instructions, "create a clearer contour with one ascent, descent, or plateau shape")
		case strings.Contains(problem, "density"):
			instructions = append(instructions, "rebalance active steps so section density follows the planned arc")
		case strings.Contains(problem, "arc") || strings.Contains(problem, "tension"):
			instructions = append(instructions, "make bars 9-12 the tension point and bars 13-16 the resolution or final peak")
		default:
			instructions = append(instructions, "make a minimal musical correction for: "+problem)
		}
	}
	if patternType == "melody" {
		instructions = append(instructions, "keep the melody sparse and preserve one memorable hook note")
	}
	return instructions
}

func planAlignmentWarnings(spec *schema.PatternSpec, _ MusicalPlan, score schema.MusicalScoreReport) []string {
	var warnings []string
	if len(score.SectionDensities) >= 4 {
		if score.SectionDensities[0] == 0 {
			warnings = append(warnings, "bars 1-4 do not introduce an active motif")
		}
		if score.SectionDensities[2] < score.SectionDensities[0] && spec.PatternType != "melody" {
			warnings = append(warnings, "bars 9-12 do not increase or hold enough tension")
		}
		if score.SectionDensities[3] == 0 {
			warnings = append(warnings, "bars 13-16 lack a resolution or final peak")
		}
	}
	return warnings
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
