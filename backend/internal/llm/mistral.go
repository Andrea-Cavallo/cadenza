package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// MistralProvider implements the Provider interface for Mistral AI API.
// Mistral API is OpenAI-compatible with native JSON schema support.
type MistralProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewMistralProvider creates a new Mistral provider.
// Reads API key from MISTRAL_API_KEY environment variable.
func NewMistralProvider(model string) (*MistralProvider, error) {
	apiKey := os.Getenv("MISTRAL_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("MISTRAL_API_KEY environment variable not set")
	}

	if model == "" {
		model = "mistral-large-latest"
	}

	return &MistralProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.mistral.ai/v1",
		client:  &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (p *MistralProvider) Name() string {
	return "mistral"
}

type mistralRequest struct {
	Model          string                `json:"model"`
	Messages       []mistralMessage      `json:"messages"`
	Temperature    float64               `json:"temperature"`
	MaxTokens      int                   `json:"max_tokens,omitempty"`
	ResponseFormat *mistralResponseFormat `json:"response_format,omitempty"`
}

type mistralMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mistralResponseFormat struct {
	Type       string             `json:"type"`
	JSONSchema *mistralJSONSchema `json:"json_schema,omitempty"`
}

type mistralJSONSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type mistralResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (p *MistralProvider) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	start := time.Now()

	messages := []mistralMessage{
		{Role: "system", Content: req.System},
	}
	for _, msg := range req.Messages {
		messages = append(messages, mistralMessage(msg))
	}

	var responseFormat *mistralResponseFormat
	if len(req.SchemaProperties) > 0 {
		schema := map[string]any{
			"type":                 "object",
			"properties":           req.SchemaProperties,
			"required":             req.SchemaRequired,
			"additionalProperties": false,
		}
		responseFormat = &mistralResponseFormat{
			Type: "json_schema",
			JSONSchema: &mistralJSONSchema{
				Name:   "pattern_spec",
				Strict: true,
				Schema: schema,
			},
		}
	}

	reqBody := mistralRequest{
		Model:          p.model,
		Messages:       messages,
		Temperature:    req.Temperature,
		MaxTokens:      req.MaxTokens,
		ResponseFormat: responseFormat,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	slog.Debug("mistral request", "model", p.model, "messages", len(messages))

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return GenerateResponse{}, fmt.Errorf("mistral api error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var apiResp mistralResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return GenerateResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return GenerateResponse{}, fmt.Errorf("mistral error: %s (%s)", apiResp.Error.Message, apiResp.Error.Type)
	}

	if len(apiResp.Choices) == 0 {
		return GenerateResponse{}, fmt.Errorf("no choices in response")
	}

	content := apiResp.Choices[0].Message.Content
	latency := time.Since(start)

	slog.Info("mistral response", "tokens", apiResp.Usage.TotalTokens, "latency_ms", latency.Milliseconds())

	return GenerateResponse{
		RawJSON:  []byte(content),
		Provider: p.Name(),
		Tokens:   apiResp.Usage.TotalTokens,
		Latency:  latency,
	}, nil
}
