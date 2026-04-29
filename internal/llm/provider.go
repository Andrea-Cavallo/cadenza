package llm

import (
	"context"
	"time"
)

type Provider interface {
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
	Name() string
}

type GenerateRequest struct {
	System       string
	Messages     []Message
	OutputSchema []byte
	Temperature  float64
	MaxTokens    int
}

type Message struct {
	Role    string
	Content string
}

type GenerateResponse struct {
	RawJSON  []byte
	Provider string
	Tokens   int
	Latency  time.Duration
}
