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
		fmt.Sscanf(errMsg, "section %d (bars %d-%d", &section, &fromBar, &toBar)

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

	// Range violation example
	if strings.Contains(errMsg, "out of") && strings.Contains(errMsg, "range") {
		return `<correction_example>
Ensure all MIDI notes are within the allowed range for this pattern type.
Correct step: {"active": true, "note": "A2"}
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
