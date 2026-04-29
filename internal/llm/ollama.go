package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
	timeout time.Duration
}

func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	return NewOllamaProviderWithTimeout(baseURL, model, 90*time.Second)
}

func NewOllamaProviderWithTimeout(baseURL, model string, timeout time.Duration) *OllamaProvider {
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		timeout: timeout,
		client:  &http.Client{Timeout: timeout},
	}
}

func (o *OllamaProvider) Name() string {
	return "ollama/" + o.model
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMsg    `json:"messages"`
	Stream   bool           `json:"stream"`
	Format   json.RawMessage `json:"format,omitempty"`
	Options  ollamaOptions  `json:"options,omitempty"`
}

type ollamaMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
}

type ollamaResponse struct {
	Message ollamaMsg `json:"message"`
}

func (o *OllamaProvider) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	start := time.Now()

	var msgs []ollamaMsg
	if req.System != "" {
		msgs = append(msgs, ollamaMsg{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		msgs = append(msgs, ollamaMsg{Role: m.Role, Content: m.Content})
	}

	format := buildOllamaFormat(req.OutputSchema)

	body := ollamaRequest{
		Model:    o.model,
		Messages: msgs,
		Stream:   false,
		Format:   format,
		Options:  ollamaOptions{Temperature: req.Temperature},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/chat", bytes.NewReader(bodyBytes))
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return GenerateResponse{}, fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(respBytes))
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBytes, &ollamaResp); err != nil {
		return GenerateResponse{}, fmt.Errorf("parse response: %w", err)
	}

	return GenerateResponse{
		RawJSON:  []byte(ollamaResp.Message.Content),
		Provider: "ollama/" + o.model,
		Latency:  time.Since(start),
	}, nil
}

func buildOllamaFormat(outputSchema []byte) json.RawMessage {
	if len(outputSchema) == 0 {
		return json.RawMessage(`"json"`)
	}

	var example map[string]any
	if err := json.Unmarshal(outputSchema, &example); err != nil {
		return json.RawMessage(`"json"`)
	}

	schema := map[string]any{
		"type":       "object",
		"properties": buildOllamaSchemaProps(example),
		"required":   objectKeys(example),
	}

	b, err := json.Marshal(schema)
	if err != nil {
		return json.RawMessage(`"json"`)
	}
	return b
}

func buildOllamaSchemaProps(obj map[string]any) map[string]any {
	props := make(map[string]any)
	for key, val := range obj {
		props[key] = inferSchemaProperty(val)
	}
	return props
}

func objectKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
