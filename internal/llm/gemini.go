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

// GeminiProvider implements the Provider interface for Google Gemini API.
// Uses Gemini 2.0 Flash in JSON mode.
// REFACTOR.md point 16
type GeminiProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewGeminiProvider creates a new Gemini provider.
// Reads API key from GEMINI_API_KEY environment variable.
func NewGeminiProvider(model string) (*GeminiProvider, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	if model == "" {
		model = "gemini-2.0-flash-exp"
	}

	return &GeminiProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		client:  &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

type geminiRequest struct {
	Contents          []geminiContent          `json:"contents"`
	SystemInstruction *geminiSystemInstruction `json:"systemInstruction,omitempty"`
	GenerationConfig  geminiGenerationConfig   `json:"generationConfig"`
}

type geminiSystemInstruction struct {
	Parts []geminiPart `json:"parts"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature      float64               `json:"temperature"`
	MaxOutputTokens  int                   `json:"maxOutputTokens,omitempty"`
	ResponseMIMEType string                `json:"responseMimeType,omitempty"`
	ResponseSchema   *geminiResponseSchema `json:"responseSchema,omitempty"`
}

type geminiResponseSchema struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		TotalTokenCount int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

func (p *GeminiProvider) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	start := time.Now()

	// Build contents: user messages
	var contents []geminiContent
	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: msg.Content}},
		})
	}

	// System instruction as separate field
	var sysInstruction *geminiSystemInstruction
	if req.System != "" {
		sysInstruction = &geminiSystemInstruction{
			Parts: []geminiPart{{Text: req.System}},
		}
	}

	// Build JSON schema for response
	var responseSchema *geminiResponseSchema
	if len(req.SchemaProperties) > 0 {
		responseSchema = &geminiResponseSchema{
			Type:       "object",
			Properties: req.SchemaProperties,
			Required:   req.SchemaRequired,
		}
	}

	genConfig := geminiGenerationConfig{
		Temperature:      req.Temperature,
		MaxOutputTokens:  req.MaxTokens,
		ResponseMIMEType: "application/json",
		ResponseSchema:   responseSchema,
	}

	reqBody := geminiRequest{
		Contents:          contents,
		SystemInstruction: sysInstruction,
		GenerationConfig:  genConfig,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", p.baseURL, p.model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

	slog.Debug("gemini request", "model", p.model, "messages", len(contents))

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
		return GenerateResponse{}, fmt.Errorf("gemini api error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var apiResp geminiResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return GenerateResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return GenerateResponse{}, fmt.Errorf("gemini error (code %d): %s", apiResp.Error.Code, apiResp.Error.Message)
	}

	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return GenerateResponse{}, fmt.Errorf("no content in response")
	}

	content := apiResp.Candidates[0].Content.Parts[0].Text
	latency := time.Since(start)

	slog.Info("gemini response", "tokens", apiResp.UsageMetadata.TotalTokenCount, "latency_ms", latency.Milliseconds())

	return GenerateResponse{
		RawJSON:  []byte(content),
		Provider: p.Name(),
		Tokens:   apiResp.UsageMetadata.TotalTokenCount,
		Latency:  latency,
	}, nil
}
