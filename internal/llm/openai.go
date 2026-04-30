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

// OpenAIProvider implements the Provider interface for OpenAI API.
// Uses gpt-4o structured output mode with JSON Schema.
// REFACTOR.md point 16
type OpenAIProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider.
// Reads API key from OPENAI_API_KEY environment variable.
func NewOpenAIProvider(model string) (*OpenAIProvider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	if model == "" {
		model = "gpt-4o"
	}

	return &OpenAIProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.openai.com/v1",
		client:  &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

type openAIRequest struct {
	Model          string                `json:"model"`
	Messages       []openAIMessage       `json:"messages"`
	Temperature    float64               `json:"temperature"`
	MaxTokens      int                   `json:"max_tokens,omitempty"`
	ResponseFormat *openAIResponseFormat `json:"response_format,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseFormat struct {
	Type       string            `json:"type"`
	JSONSchema *openAIJSONSchema `json:"json_schema,omitempty"`
}

type openAIJSONSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type openAIResponse struct {
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

func (p *OpenAIProvider) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	start := time.Now()

	// Build messages: system + user
	messages := []openAIMessage{
		{Role: "system", Content: req.System},
	}
	for _, msg := range req.Messages {
		messages = append(messages, openAIMessage(msg))
	}

	// Build structured output schema
	var responseFormat *openAIResponseFormat
	if len(req.SchemaProperties) > 0 {
		schema := map[string]any{
			"type":                 "object",
			"properties":           req.SchemaProperties,
			"required":             req.SchemaRequired,
			"additionalProperties": false,
		}
		responseFormat = &openAIResponseFormat{
			Type: "json_schema",
			JSONSchema: &openAIJSONSchema{
				Name:   "pattern_spec",
				Strict: true,
				Schema: schema,
			},
		}
	}

	reqBody := openAIRequest{
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

	slog.Debug("openai request", "model", p.model, "messages", len(messages))

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return GenerateResponse{}, fmt.Errorf("openai api error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return GenerateResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return GenerateResponse{}, fmt.Errorf("openai error: %s (%s)", apiResp.Error.Message, apiResp.Error.Type)
	}

	if len(apiResp.Choices) == 0 {
		return GenerateResponse{}, fmt.Errorf("no choices in response")
	}

	content := apiResp.Choices[0].Message.Content
	latency := time.Since(start)

	slog.Info("openai response", "tokens", apiResp.Usage.TotalTokens, "latency_ms", latency.Milliseconds())

	return GenerateResponse{
		RawJSON:  []byte(content),
		Provider: p.Name(),
		Tokens:   apiResp.Usage.TotalTokens,
		Latency:  latency,
	}, nil
}
