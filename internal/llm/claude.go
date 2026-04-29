package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
)

type ClaudeProvider struct {
	client *anthropic.Client
	model  string
}

func NewClaudeProvider(model string) (*ClaudeProvider, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &ClaudeProvider{client: &client, model: model}, nil
}

func (c *ClaudeProvider) Name() string {
	return "claude/" + c.model
}

func (c *ClaudeProvider) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	start := time.Now()

	var messages []anthropic.MessageParam
	for _, m := range req.Messages {
		switch m.Role {
		case "user":
			messages = append(messages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(m.Content),
			))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(
				anthropic.NewTextBlock(m.Content),
			))
		}
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: int64(req.MaxTokens),
		Messages:  messages,
	}

	if req.System != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: req.System},
		}
	}

	if req.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Temperature)
	}

	if len(req.OutputSchema) > 0 {
		schemaProps, required := buildSchemaFromJSON(req.OutputSchema)
		params.Tools = []anthropic.ToolUnionParam{{
			OfTool: &anthropic.ToolParam{
				Name:        "generate_pattern",
				Description: param.Opt[string]{Value: "Generate a musical pattern as structured JSON matching PatternSpec schema"},
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: schemaProps,
					Required:   required,
				},
			},
		}}
		params.ToolChoice = anthropic.ToolChoiceParamOfTool("generate_pattern")
	}

	resp, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("claude api: %w", err)
	}

	content := extractToolUseInput(resp)
	if content == "" {
		content = extractTextContent(resp)
	}

	return GenerateResponse{
		RawJSON:  []byte(content),
		Provider: "claude",
		Tokens:   int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		Latency:  time.Since(start),
	}, nil
}

func extractToolUseInput(resp *anthropic.Message) string {
	for _, block := range resp.Content {
		if block.Type == "tool_use" {
			if len(block.Input) > 0 {
				return string(block.Input)
			}
		}
	}
	return ""
}

func extractTextContent(resp *anthropic.Message) string {
	for _, block := range resp.Content {
		if block.Type == "text" {
			return block.Text
		}
	}
	return ""
}

func buildSchemaFromJSON(exampleJSON []byte) (map[string]any, []string) {
	var example map[string]any
	if err := json.Unmarshal(exampleJSON, &example); err != nil {
		return map[string]any{}, nil
	}

	properties := make(map[string]any)
	var required []string

	for key, val := range example {
		required = append(required, key)
		properties[key] = inferSchemaProperty(val)
	}

	return properties, required
}

func inferSchemaProperty(val any) map[string]any {
	switch v := val.(type) {
	case string:
		return map[string]any{"type": "string"}
	case float64:
		return map[string]any{"type": "number"}
	case bool:
		return map[string]any{"type": "boolean"}
	case []any:
		if len(v) > 0 {
			return map[string]any{"type": "array", "items": inferSchemaProperty(v[0])}
		}
		return map[string]any{"type": "array"}
	case map[string]any:
		props := make(map[string]any)
		var req []string
		for k, inner := range v {
			props[k] = inferSchemaProperty(inner)
			req = append(req, k)
		}
		return map[string]any{"type": "object", "properties": props, "required": req}
	default:
		return map[string]any{"type": "string"}
	}
}
