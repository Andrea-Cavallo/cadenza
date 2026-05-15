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

// GroqProvider implements the Provider interface for Groq API.
// Groq provides ultra-fast inference on LPU hardware with an OpenAI-compatible API.
type GroqProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewGroqProvider creates a new Groq provider.
// Reads API key from GROQ_API_KEY environment variable.
func NewGroqProvider(model string) (*GroqProvider, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GROQ_API_KEY environment variable not set")
	}

	if model == "" {
		model = "llama-3.3-70b-versatile"
	}

	return &GroqProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.groq.com/openai/v1",
		client:  &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (p *GroqProvider) Name() string {
	return "groq"
}

type groqRequest struct {
	Model          string              `json:"model"`
	Messages       []groqMessage       `json:"messages"`
	Temperature    float64             `json:"temperature"`
	MaxTokens      int                 `json:"max_tokens,omitempty"`
	ResponseFormat *groqResponseFormat `json:"response_format,omitempty"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponseFormat struct {
	Type       string          `json:"type"`
	JSONSchema *groqJSONSchema `json:"json_schema,omitempty"`
}

type groqJSONSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type groqResponse struct {
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

func (p *GroqProvider) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	start := time.Now()

	messages := []groqMessage{
		{Role: "system", Content: req.System},
	}
	for _, msg := range req.Messages {
		messages = append(messages, groqMessage(msg))
	}

	var responseFormat *groqResponseFormat
	if len(req.SchemaProperties) > 0 {
		schema := map[string]any{
			"type":                 "object",
			"properties":           req.SchemaProperties,
			"required":             req.SchemaRequired,
			"additionalProperties": false,
		}
		responseFormat = &groqResponseFormat{
			Type: "json_schema",
			JSONSchema: &groqJSONSchema{
				Name:   "pattern_spec",
				Strict: true,
				Schema: schema,
			},
		}
	}

	reqBody := groqRequest{
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

	slog.Debug("groq request", "model", p.model, "messages", len(messages))

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
		return GenerateResponse{}, fmt.Errorf("groq api error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var apiResp groqResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return GenerateResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return GenerateResponse{}, fmt.Errorf("groq error: %s (%s)", apiResp.Error.Message, apiResp.Error.Type)
	}

	if len(apiResp.Choices) == 0 {
		return GenerateResponse{}, fmt.Errorf("no choices in response")
	}

	content := apiResp.Choices[0].Message.Content
	latency := time.Since(start)

	slog.Info("groq response", "tokens", apiResp.Usage.TotalTokens, "latency_ms", latency.Milliseconds())

	return GenerateResponse{
		RawJSON:  []byte(content),
		Provider: p.Name(),
		Tokens:   apiResp.Usage.TotalTokens,
		Latency:  latency,
	}, nil
}
