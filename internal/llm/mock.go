package llm

import (
	"context"
	"time"
)

type MockProvider struct {
	Response []byte
	Err      error
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) Generate(_ context.Context, _ GenerateRequest) (GenerateResponse, error) {
	if m.Err != nil {
		return GenerateResponse{}, m.Err
	}
	return GenerateResponse{
		RawJSON:  m.Response,
		Provider: "mock",
		Tokens:   100,
		Latency:  10 * time.Millisecond,
	}, nil
}
