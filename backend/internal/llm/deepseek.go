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

// DeepSeekProvider implements the Provider interface for DeepSeek API.
// DeepSeek API is OpenAI-compatible, using the same /v1/chat/completions endpoint
// with Bearer token auth. Uses response_format json_object (not json_schema,
// which DeepSeek does not support). Schema enforcement is done via prompt.
type DeepSeekProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewDeepSeekProvider creates a new DeepSeek provider.
// Reads API key from DEEPSEEK_API_KEY environment variable.
func NewDeepSeekProvider(model string) (*DeepSeekProvider, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY environment variable not set")
	}

	if model == "" {
		model = "deepseek-chat"
	}

	return &DeepSeekProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.deepseek.com/v1",
		client:  &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (p *DeepSeekProvider) Name() string {
	return "deepseek"
}

type deepSeekRequest struct {
	Model          string                   `json:"model"`
	Messages       []deepSeekMessage        `json:"messages"`
	Temperature    float64                  `json:"temperature"`
	MaxTokens      int                      `json:"max_tokens,omitempty"`
	ResponseFormat *deepSeekResponseFormat  `json:"response_format,omitempty"`
}

type deepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepSeekResponseFormat struct {
	Type string `json:"type"`
}

type deepSeekResponse struct {
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

func (p *DeepSeekProvider) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	start := time.Now()

	messages := []deepSeekMessage{
		{Role: "system", Content: req.System},
	}
	for _, msg := range req.Messages {
		messages = append(messages, deepSeekMessage(msg))
	}

	var responseFormat *deepSeekResponseFormat
	if len(req.SchemaProperties) > 0 {
		responseFormat = &deepSeekResponseFormat{
			Type: "json_object",
		}
	}

	reqBody := deepSeekRequest{
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

	slog.Debug("deepseek request", "model", p.model, "messages", len(messages))

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
		return GenerateResponse{}, fmt.Errorf("deepseek api error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var apiResp deepSeekResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return GenerateResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return GenerateResponse{}, fmt.Errorf("deepseek error: %s (%s)", apiResp.Error.Message, apiResp.Error.Type)
	}

	if len(apiResp.Choices) == 0 {
		return GenerateResponse{}, fmt.Errorf("no choices in response")
	}

	content := apiResp.Choices[0].Message.Content
	latency := time.Since(start)

	slog.Info("deepseek response", "tokens", apiResp.Usage.TotalTokens, "latency_ms", latency.Milliseconds())

	return GenerateResponse{
		RawJSON:  []byte(content),
		Provider: p.Name(),
		Tokens:   apiResp.Usage.TotalTokens,
		Latency:  latency,
	}, nil
}
