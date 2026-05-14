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

// RetryExhaustedError is returned when all retry attempts are consumed.
// It carries the last invalid JSON response so callers can attempt programmatic repair.
type RetryExhaustedError struct {
	LastRaw     []byte
	TotalTokens int
	Cause       error
}

func (e *RetryExhaustedError) Error() string {
	return fmt.Sprintf("max retries exceeded (tokens used: %d): %v", e.TotalTokens, e.Cause)
}

func (e *RetryExhaustedError) Unwrap() error { return e.Cause }

func GenerateWithRetry(ctx context.Context, p Provider, req GenerateRequest, validate ValidateFunc) ([]byte, error) {
	var lastErr error
	var totalTokens int
	var lastRaw []byte

	retries := maxRetries
	if req.MaxRetries > 0 {
		retries = req.MaxRetries
	}

	for attempt := 0; attempt < retries; attempt++ {
		slog.Debug("llm attempt starting", "attempt", attempt+1, "max", retries, "provider", p.Name())

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
		slog.Debug("llm raw response", "attempt", attempt+1, "bytes", len(resp.RawJSON), "snippet", jsonSnippet(resp.RawJSON, 500))

		if err := validate(resp.RawJSON); err == nil {
			slog.Info("llm generation complete", "total_tokens", totalTokens, "attempts", attempt+1)
			return resp.RawJSON, nil
		} else {
			lastErr = err
			lastRaw = resp.RawJSON
		}

		isStructural := isStructuralError(resp.RawJSON)
		slog.Warn("llm output invalid, retrying",
			"attempt", attempt+1,
			"structural", isStructural,
			"error", lastErr,
		)
		// Log each validation error on its own line for easier reading.
		for i, e := range splitValidationErrors(lastErr.Error()) {
			slog.Debug("validation error", "attempt", attempt+1, "index", i, "detail", e)
		}

		req.Messages = appendCorrection(req.Messages, req.OutputSchema, resp.RawJSON, lastErr, isStructural)
		slog.Debug("correction appended", "attempt", attempt+1, "structural", isStructural, "example_type", correctionExampleType(lastErr.Error()))

		backoff := time.Duration(initialBackoffMs*(1<<attempt)) * time.Millisecond
		slog.Debug("backoff before next attempt", "attempt", attempt+1, "backoff_ms", backoff.Milliseconds())
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}

	slog.Warn("all retries exhausted", "total_tokens", totalTokens, "last_error", lastErr)
	return nil, &RetryExhaustedError{LastRaw: lastRaw, TotalTokens: totalTokens, Cause: lastErr}
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

	// Parse chord coherence failures and build correction examples with the actual chord tones.
	// Error format: "section 2 (bars 5-8): chord coherence 33% < required 75% for bassline (found 1/3 chord tones of Bm; chord tones: B, D, F#)"
	if strings.Contains(errMsg, "chord coherence") && strings.Contains(errMsg, "section") {
		idx := strings.Index(errMsg, "section ")
		if idx < 0 {
			return ""
		}
		sub := errMsg[idx:]
		var section, fromBar, toBar int
		n, _ := fmt.Sscanf(sub, "section %d (bars %d-%d", &section, &fromBar, &toBar)
		if n < 3 {
			section, fromBar, toBar = 1, 1, 4
		}

		// Choose octave based on pattern type in error message
		octave := 2 // bassline default (A1-G3)
		if strings.Contains(errMsg, "for arpeggio") {
			octave = 4 // arpeggio (C3-C6)
		} else if strings.Contains(errMsg, "for melody") {
			octave = 5 // melody (C4-C7)
		}

		// Build examples from the actual chord tones embedded in the error message
		var noteLines string
		const marker = "; chord tones: "
		if ctIdx := strings.Index(sub, marker); ctIdx >= 0 {
			rest := sub[ctIdx+len(marker):]
			if endIdx := strings.IndexAny(rest, ")\n"); endIdx >= 0 {
				rest = rest[:endIdx]
			}
			parts := strings.Split(rest, ", ")
			var lines []string
			for i, ct := range parts {
				ct = strings.TrimSpace(ct)
				if ct == "" {
					continue
				}
				if i == 0 {
					lines = append(lines, fmt.Sprintf(`{"active": true, "note": "%s%d", "accent": true}`, ct, octave))
				} else {
					lines = append(lines, fmt.Sprintf(`{"active": true, "note": "%s%d"}`, ct, octave))
				}
			}
			lines = append(lines, `{"active": false}`)
			noteLines = strings.Join(lines, "\n")
		} else {
			// Fallback when chord tones are not in the error (older format)
			noteLines = fmt.Sprintf(
				`{"active": true, "note": "A%d", "accent": true}`+"\n"+
					`{"active": true, "note": "E%d"}`+"\n"+
					`{"active": true, "note": "C%d"}`+"\n"+
					`{"active": false}`,
				octave, octave+1, octave+1)
		}

		return fmt.Sprintf(`<correction_example>
For section %d (bars %d-%d), ensure MOST steps use chord tones from the progression.
Example of correct steps using the actual chord tones:
%s
Do NOT use notes from a different chord section here.
</correction_example>
`, section, fromBar, toBar, noteLines)
	}

	// Scale violation example — extract the wrong note and scale from the error message.
	// Example error: "step[5] note F4 not in A dorian scale"
	if strings.Contains(errMsg, "not in") && strings.Contains(errMsg, "scale") {
		wrongNote := extractNoteFromScaleError(errMsg)
		scaleName := extractScaleFromError(errMsg)
		scaleHint := ""
		if scaleName != "" {
			scaleHint = fmt.Sprintf("\nThe scale is %s. Check the scale notes provided at the top of this prompt and use ONLY those note names.", scaleName)
		}
		wrongNoteExample := wrongNote
		if wrongNoteExample == "" {
			wrongNoteExample = "A#3"
		}
		return fmt.Sprintf(`<correction_example>
Ensure all notes belong to the specified scale. Note %s is not valid.%s
Common mistake: confusing F with F# (or Bb with B, Eb with E, etc.) — always check the scale note list.
</correction_example>
`, wrongNoteExample, scaleHint)
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

// jsonSnippet returns up to maxLen bytes of raw JSON as a string, appending "…" if truncated.
func jsonSnippet(raw []byte, maxLen int) string {
	if len(raw) <= maxLen {
		return string(raw)
	}
	return string(raw[:maxLen]) + "…"
}

// splitValidationErrors splits a "validation failed:\n- err1\n- err2" string into individual items.
func splitValidationErrors(msg string) []string {
	const prefix = "validation failed:\n- "
	if idx := strings.Index(msg, prefix); idx >= 0 {
		msg = msg[idx+len(prefix):]
	}
	parts := strings.Split(msg, "\n- ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// correctionExampleType returns a short label describing which correction branch was matched.
func correctionExampleType(errMsg string) string {
	switch {
	case strings.Contains(errMsg, "chord coherence"):
		return "chord_coherence"
	case strings.Contains(errMsg, "not in") && strings.Contains(errMsg, "scale"):
		return "scale_violation"
	case strings.Contains(errMsg, "out of") && strings.Contains(errMsg, "range"):
		return "range_violation"
	case strings.Contains(errMsg, "unknown action"):
		return "evolution_action"
	case strings.Contains(errMsg, "intensity"):
		return "evolution_intensity"
	case strings.Contains(errMsg, "density"):
		return "density"
	default:
		return "generic"
	}
}

// extractScaleFromError parses a scale name from a validation error message like
// "step[5] note F4 not in A dorian scale" → "A dorian".
func extractScaleFromError(errMsg string) string {
	const marker = " not in "
	idx := strings.Index(errMsg, marker)
	if idx < 0 {
		return ""
	}
	rest := errMsg[idx+len(marker):]
	end := strings.Index(rest, " scale")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}

// extractNoteFromScaleError parses the invalid note from a validation error message like
// "step[5] note F4 not in A dorian scale" → "F4".
func extractNoteFromScaleError(errMsg string) string {
	const noteMarker = " note "
	const notInMarker = " not in "
	start := strings.Index(errMsg, noteMarker)
	if start < 0 {
		return ""
	}
	rest := errMsg[start+len(noteMarker):]
	end := strings.Index(rest, notInMarker)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}
