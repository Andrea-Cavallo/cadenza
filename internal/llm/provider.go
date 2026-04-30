package llm

import (
	"context"
	"time"
)

// Provider is the LLM backend interface (Claude, Ollama, OpenAI, Gemini).
type Provider interface {
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
	Name() string
}

// GenerateRequest encapsulates a pattern generation request to an LLM provider.
type GenerateRequest struct {
	System           string
	Messages         []Message
	OutputSchema     []byte
	SchemaProperties map[string]any // REFACTOR.md point 1: Pre-generated schema properties from invopop/jsonschema
	SchemaRequired   []string       // REFACTOR.md point 1: Pre-generated required fields list
	Temperature      float64
	MaxTokens        int
}

// Message is a single chat message (role + content) in the LLM conversation.
type Message struct {
	Role    string
	Content string
}

// GenerateResponse holds raw JSON output and metadata from an LLM call.
type GenerateResponse struct {
	RawJSON  []byte
	Provider string
	Tokens   int
	Latency  time.Duration
}
