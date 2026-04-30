package llm

import (
	"context"
	"errors"
	"testing"
)

func TestMockProviderName(t *testing.T) {
	m := &MockProvider{}
	if m.Name() != "mock" {
		t.Fatalf("expected mock, got %q", m.Name())
	}
}

func TestMockProviderGenerate(t *testing.T) {
	data := []byte(`{"pattern":"test"}`)
	m := &MockProvider{Response: data}

	resp, err := m.Generate(context.Background(), GenerateRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp.RawJSON) != string(data) {
		t.Fatalf("response mismatch: %q", resp.RawJSON)
	}
	if resp.Provider != "mock" {
		t.Fatalf("expected provider mock, got %q", resp.Provider)
	}
}

func TestMockProviderError(t *testing.T) {
	m := &MockProvider{Err: errors.New("api down")}

	_, err := m.Generate(context.Background(), GenerateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerateWithRetrySuccess(t *testing.T) {
	validJSON := []byte(`{"spec_version":"1.0","pattern_type":"bassline"}`)
	m := &MockProvider{Response: validJSON}

	validate := func(raw []byte) error { return nil }

	result, err := GenerateWithRetry(context.Background(), m, GenerateRequest{}, validate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != string(validJSON) {
		t.Fatalf("result mismatch: %q", result)
	}
}

func TestGenerateWithRetryProviderError(t *testing.T) {
	m := &MockProvider{Err: errors.New("network timeout")}

	validate := func(raw []byte) error { return nil }

	_, err := GenerateWithRetry(context.Background(), m, GenerateRequest{}, validate)
	if err == nil {
		t.Fatal("expected error from provider failure")
	}
}

func TestGenerateWithRetryValidationFailure(t *testing.T) {
	invalidJSON := []byte(`{"spec_version":"1.0","pattern_type":"bad"}`)
	m := &MockProvider{Response: invalidJSON}

	validate := func(raw []byte) error {
		return errors.New("density too low")
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately to avoid backoff delay in test
	cancel()

	_, err := GenerateWithRetry(ctx, m, GenerateRequest{}, validate)
	if err == nil {
		t.Fatal("expected error after max retries or context cancel")
	}
}

func TestGenerateWithRetryContextCancel(t *testing.T) {
	// Provider returns valid JSON but validation always fails
	data := []byte(`{"spec_version":"1.0"}`)
	m := &MockProvider{Response: data}

	validate := func(raw []byte) error {
		return errors.New("always fails")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := GenerateWithRetry(ctx, m, GenerateRequest{}, validate)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestIsStructuralError(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{"valid JSON", []byte(`{"key":"value"}`), false},
		{"invalid JSON", []byte(`not json at all`), true},
		{"empty", []byte(``), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isStructuralError(tt.input)
			if got != tt.expected {
				t.Errorf("isStructuralError(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAppendCorrection(t *testing.T) {
	msgs := []Message{{Role: "user", Content: "generate pattern"}}
	schema := []byte(`{"type":"object"}`)
	invalid := []byte(`{"bad":"data"}`)
	valErr := errors.New("density too low")

	result := appendCorrection(msgs, schema, invalid, valErr, false)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[1].Role != "user" {
		t.Fatalf("correction should be user role, got %q", result[1].Role)
	}
}

func TestAppendCorrectionStructural(t *testing.T) {
	msgs := []Message{{Role: "user", Content: "generate"}}
	invalid := []byte(`invalid json`)
	valErr := errors.New("json parse error")

	result := appendCorrection(msgs, nil, invalid, valErr, true)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
}
