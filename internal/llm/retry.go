package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	maxRetries        = 3
	initialBackoffMs  = 500
	backoffMultiplier = 2
)

// ValidateFunc checks raw JSON output for musical constraint violations.
type ValidateFunc func(raw []byte) error

func GenerateWithRetry(ctx context.Context, p Provider, req GenerateRequest, validate ValidateFunc) ([]byte, error) {
	var lastErr error
	var totalTokens int

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := p.Generate(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("llm call: %w", err)
		}

		totalTokens += resp.Tokens
		slog.Info("llm response",
			"provider", resp.Provider,
			"tokens", resp.Tokens,
			"latency", resp.Latency.Round(time.Millisecond),
			"attempt", attempt+1,
		)

		if err := validate(resp.RawJSON); err == nil {
			slog.Info("llm generation complete", "total_tokens", totalTokens, "attempts", attempt+1)
			return resp.RawJSON, nil
		} else {
			lastErr = err
		}

		isStructural := isStructuralError(resp.RawJSON)
		req.Messages = appendCorrection(req.Messages, req.OutputSchema, resp.RawJSON, lastErr, isStructural)
		slog.Warn("llm output invalid, retrying",
			"attempt", attempt+1,
			"structural", isStructural,
			"error", lastErr,
		)

		backoff := time.Duration(initialBackoffMs*(1<<attempt)) * time.Millisecond
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}

	return nil, fmt.Errorf("max retries exceeded (tokens used: %d): %w", totalTokens, lastErr)
}

func isStructuralError(raw []byte) bool {
	var obj map[string]any
	return json.Unmarshal(raw, &obj) != nil
}

func appendCorrection(msgs []Message, outputSchema, invalidJSON []byte, valErr error, structural bool) []Message {
	schemaBlock := ""
	if len(outputSchema) > 0 {
		schemaBlock = fmt.Sprintf("<json_schema>\n%s\n</json_schema>\n", string(outputSchema))
	}

	var instruction string
	if structural {
		instruction = "Your previous response was NOT valid JSON. Return ONLY a valid JSON object matching the schema exactly. No markdown fences, no explanation, no text before or after the JSON."
	} else {
		instruction = "Your previous response was valid JSON but failed musical validation. Fix the specific issues listed below while keeping the rest of the pattern intact. Return ONLY the corrected JSON."
	}

	// REFACTOR.md point 3: Include positive example in correction prompt
	// Extract a corrective example from the error message if possible
	exampleBlock := extractCorrectionExample(valErr)

	correction := fmt.Sprintf(
		"%s<previous_invalid_output>\n%s\n</previous_invalid_output>\n"+
			"<validation_errors>\n%s\n</validation_errors>\n"+
			"%s"+
			"<retry_instruction>%s</retry_instruction>",
		schemaBlock,
		string(invalidJSON),
		valErr.Error(),
		exampleBlock,
		instruction,
	)
	return append(msgs, Message{Role: "user", Content: correction})
}

// extractCorrectionExample generates a positive example for common validation failures.
// REFACTOR.md point 3: Show correct step format for the failing section.
func extractCorrectionExample(valErr error) string {
	errMsg := valErr.Error()

	// Parse section number and chord from error message patterns
	// Example: "section 2 (bars 5-8): chord coherence 50% < required 75% for bassline (found 2/4 chord tones of Fm)"
	if strings.Contains(errMsg, "chord coherence") && strings.Contains(errMsg, "section") {
		// Extract section info for example
		var section, fromBar, toBar int
		_, _ = fmt.Sscanf(errMsg, "section %d (bars %d-%d", &section, &fromBar, &toBar)

		// Simple example showing correct usage
		return fmt.Sprintf(`<correction_example>
For section %d (bars %d-%d), ensure most notes are chord tones.
Example of correct steps using chord tones:
{"active": true, "note": "F3", "accent": true}
{"active": true, "note": "Ab3"}
{"active": true, "note": "C4"}
{"active": false}
</correction_example>
`, section, fromBar, toBar)
	}

	// Scale violation example
	if strings.Contains(errMsg, "not in") && strings.Contains(errMsg, "scale") {
		return `<correction_example>
Ensure all notes belong to the specified scale.
Example: For Am natural minor scale, use: A B C D E F G (no sharps/flats except those in the key signature)
Correct step: {"active": true, "note": "A3"}
Incorrect step: {"active": true, "note": "A#3"}
</correction_example>
`
	}

	// Range violation example — pattern-type-specific octave guidance
	if strings.Contains(errMsg, "out of") && strings.Contains(errMsg, "range") {
		switch {
		case strings.Contains(errMsg, "bassline range"):
			return `<correction_example>
Bassline range is MIDI 33-55 (A1 to G3). Use octave 2 as the center (A2=MIDI45, E2=MIDI40, D2=MIDI38).
NEVER use octave 3 or higher for bassline notes.
Wrong: {"active": true, "note": "A3"}  (MIDI 57 — too high)
Right: {"active": true, "note": "A2"}  (MIDI 45 — correct)
</correction_example>
`
		case strings.Contains(errMsg, "arpeggio range"):
			return `<correction_example>
Arpeggio range is MIDI 48-84 (C3 to C6). Use octaves 3-5 (e.g. A3=MIDI57, E4=MIDI64, A5=MIDI81).
NEVER exceed C6 (MIDI 84).
Wrong: {"active": true, "note": "F6"}  (MIDI 89 — too high)
Right: {"active": true, "note": "F5"}  (MIDI 77 — correct)
</correction_example>
`
		case strings.Contains(errMsg, "melody range"):
			return `<correction_example>
Melody range is MIDI 60-96 (C4 to C7). Use octaves 4-6 (e.g. A4=MIDI69, E5=MIDI76).
NEVER go below C4 (MIDI 60).
Wrong: {"active": true, "note": "A3"}  (MIDI 57 — too low)
Right: {"active": true, "note": "A4"}  (MIDI 69 — correct)
</correction_example>
`
		default:
			return `<correction_example>
Ensure all MIDI notes are within the allowed range for this pattern type.
Bassline: octave 1-3 (center A2). Arpeggio: octave 3-5 (center E4). Melody: octave 4-6 (center A4).
</correction_example>
`
		}
	}

	// Unknown evolution action
	if strings.Contains(errMsg, "unknown action") {
		return `<correction_example>
The "action" field in each evolution step must be EXACTLY one of these strings:
  introduce, build, peak, release, octave_up, octave_down,
  density_up, density_down, add_chord_note, strip_to_root, ornament

The "intensity" must be a float between 0.0 and 1.0.

Correct evolution array:
[
  {"from_bar": 1, "to_bar": 4,  "action": "introduce", "intensity": 0.3},
  {"from_bar": 5, "to_bar": 8,  "action": "build",     "intensity": 0.6},
  {"from_bar": 9, "to_bar": 12, "action": "peak",       "intensity": 0.9},
  {"from_bar": 13,"to_bar": 16, "action": "release",    "intensity": 0.5}
]
</correction_example>
`
	}

	// Intensity out of range
	if strings.Contains(errMsg, "intensity") && strings.Contains(errMsg, "out of [0, 1]") {
		return `<correction_example>
"intensity" must be a decimal float between 0.0 and 1.0 (inclusive).
Wrong: {"action": "build", "intensity": 8}
Right: {"action": "build", "intensity": 0.8}
</correction_example>
`
	}

	// Density violation example
	if strings.Contains(errMsg, "density") {
		return `<correction_example>
Adjust the number of active steps to match density requirements.
Example of balanced density:
{"active": true, "note": "A2", "accent": true}
{"active": true, "note": "A2"}
{"active": false}
{"active": true, "note": "E2", "slide": true}
</correction_example>
`
	}

	return ""
}
