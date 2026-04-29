package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

const (
	maxRetries        = 3
	initialBackoffMs  = 500
	backoffMultiplier = 2
)

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

	correction := fmt.Sprintf(
		"%s<previous_invalid_output>\n%s\n</previous_invalid_output>\n"+
			"<validation_errors>\n%s\n</validation_errors>\n"+
			"<retry_instruction>%s</retry_instruction>",
		schemaBlock,
		string(invalidJSON),
		valErr.Error(),
		instruction,
	)
	return append(msgs, Message{Role: "user", Content: correction})
}
