package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func TestProviderConstructors_EnvAndNames(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	if _, err := NewClaudeProvider("claude-opus"); err == nil {
		t.Fatal("expected missing anthropic key error")
	}

	t.Setenv("OPENAI_API_KEY", "")
	if _, err := NewOpenAIProvider(""); err == nil {
		t.Fatal("expected missing openai key error")
	}

	t.Setenv("GEMINI_API_KEY", "")
	if _, err := NewGeminiProvider(""); err == nil {
		t.Fatal("expected missing gemini key error")
	}

	ollama := NewOllamaProvider("http://localhost:11434", "qwen")
	if ollama.Name() != "ollama/qwen" {
		t.Fatalf("unexpected ollama name %q", ollama.Name())
	}

	ollama2 := NewOllamaProviderWithTimeout("http://localhost:11434", "qwen", time.Second)
	if ollama2.timeout != time.Second {
		t.Fatalf("expected timeout override")
	}

	t.Setenv("OPENAI_API_KEY", "x")
	openai, err := NewOpenAIProvider("")
	if err != nil {
		t.Fatalf("unexpected openai constructor error: %v", err)
	}
	if openai.Name() != "openai" {
		t.Fatalf("unexpected openai name %q", openai.Name())
	}

	t.Setenv("GEMINI_API_KEY", "x")
	gemini, err := NewGeminiProvider("")
	if err != nil {
		t.Fatalf("unexpected gemini constructor error: %v", err)
	}
	if gemini.Name() != "gemini" {
		t.Fatalf("unexpected gemini name %q", gemini.Name())
	}
	if openai.model != "gpt-4o" {
		t.Fatalf("expected default openai model, got %q", openai.model)
	}
	if gemini.model != "gemini-2.0-flash-exp" {
		t.Fatalf("expected default gemini model, got %q", gemini.model)
	}
}

func TestSchemaHelpers(t *testing.T) {
	example := []byte(`{"name":"bass","active":true,"notes":["A2"],"meta":{"bpm":122}}`)
	props, required := buildSchemaFromJSON(example)
	if len(props) == 0 || len(required) == 0 {
		t.Fatal("expected schema props and required fields")
	}

	meta, ok := props["meta"].(map[string]any)
	if !ok || meta["type"] != "object" {
		t.Fatalf("expected meta object schema, got %#v", props["meta"])
	}
	if inferSchemaProperty([]any{})["type"] != "array" {
		t.Fatal("empty array inference should still be array")
	}
	if inferSchemaProperty(map[string]any{"value": 1})["type"] != "object" {
		t.Fatal("map inference should be object")
	}
	if inferSchemaProperty(float64(123))["type"] != "number" {
		t.Fatal("number inference should be number")
	}
	if inferSchemaProperty(struct{}{})["type"] != "string" {
		t.Fatal("default inference should fall back to string")
	}
	props, required = buildSchemaFromJSON([]byte(`not-json`))
	if len(props) != 0 || len(required) != 0 {
		t.Fatal("invalid schema example should return empty schema")
	}
}

func TestOllamaFormatHelpers(t *testing.T) {
	schema := buildOllamaFormat(nil, nil, nil)
	if string(schema) != `"json"` {
		t.Fatalf("expected json mode fallback, got %s", string(schema))
	}

	props := map[string]any{"name": map[string]any{"type": "string"}}
	withProps := buildOllamaFormat([]byte(`{}`), props, []string{"name"})
	var decoded map[string]any
	if err := json.Unmarshal(withProps, &decoded); err != nil {
		t.Fatalf("schema with props should be valid json: %v", err)
	}
	if decoded["type"] != "object" {
		t.Fatalf("unexpected schema type: %#v", decoded)
	}

	built := buildOllamaSchemaProps(map[string]any{"enabled": true, "count": float64(2)})
	if len(built) != 2 {
		t.Fatalf("expected 2 props, got %d", len(built))
	}
	keys := objectKeys(map[string]any{"a": 1, "b": 2})
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	fromExample := buildOllamaFormat([]byte(`{"count":1}`), nil, nil)
	if !strings.Contains(string(fromExample), `"properties"`) {
		t.Fatalf("expected inferred schema, got %s", string(fromExample))
	}

	badExample := buildOllamaFormat([]byte(`not-json`), nil, nil)
	if string(badExample) != `"json"` {
		t.Fatalf("expected json fallback for invalid example, got %s", string(badExample))
	}

	badProps := buildOllamaFormat([]byte(`{}`), map[string]any{"bad": make(chan int)}, []string{"bad"})
	if string(badProps) != `"json"` {
		t.Fatalf("expected json fallback for unmarshalable schema props, got %s", string(badProps))
	}
}

func TestExtractCorrectionExample_Branches(t *testing.T) {
	chord := extractCorrectionExample(assertErr("section 2 (bars 5-8): chord coherence 50% < required 75% for bassline"))
	if !strings.Contains(chord, "section 2") {
		t.Fatalf("expected chord example, got %q", chord)
	}

	scale := extractCorrectionExample(assertErr("step[0] note A#3 not in A minor_natural scale"))
	if !strings.Contains(scale, "A#3") {
		t.Fatalf("expected scale example, got %q", scale)
	}

	rng := extractCorrectionExample(assertErr("step[0] note C7 out of bassline range"))
	if !strings.Contains(rng, "allowed range") {
		t.Fatalf("expected range example, got %q", rng)
	}

	density := extractCorrectionExample(assertErr("density too low: 1 active steps"))
	if !strings.Contains(density, "balanced density") {
		t.Fatalf("expected density example, got %q", density)
	}
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func assertErr(msg string) error {
	return simpleErr(msg)
}

func TestMainTestEnvIsClean(t *testing.T) {
	_ = os.Getenv("PATH")
}

func TestOllamaGenerate_SuccessAndRequestShape(t *testing.T) {
	var captured ollamaRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content-type %q", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"{\"pattern_type\":\"bassline\"}"}}`))
	}))
	defer server.Close()

	provider := NewOllamaProviderWithTimeout(server.URL, "qwen-test", time.Second)
	resp, err := provider.Generate(context.Background(), GenerateRequest{
		System: "system rules",
		Messages: []Message{
			{Role: "user", Content: "generate"},
		},
		SchemaProperties: map[string]any{
			"pattern_type": map[string]any{"type": "string"},
		},
		SchemaRequired: []string{"pattern_type"},
		Temperature:    0.3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.Model != "qwen-test" {
		t.Fatalf("unexpected model %q", captured.Model)
	}
	if len(captured.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(captured.Messages))
	}
	if captured.Messages[0].Role != "system" || captured.Messages[1].Role != "user" {
		t.Fatalf("unexpected roles: %#v", captured.Messages)
	}
	if resp.Provider != "ollama/qwen-test" {
		t.Fatalf("unexpected provider %q", resp.Provider)
	}
	if string(resp.RawJSON) != `{"pattern_type":"bassline"}` {
		t.Fatalf("unexpected raw json %q", string(resp.RawJSON))
	}
	if resp.Latency <= 0 {
		t.Fatalf("expected positive latency, got %v", resp.Latency)
	}
}

func TestOllamaGenerate_ErrorBranches(t *testing.T) {
	t.Run("status error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "bad request", http.StatusBadRequest)
		}))
		defer server.Close()

		provider := NewOllamaProvider(server.URL, "qwen")
		_, err := provider.Generate(context.Background(), GenerateRequest{})
		if err == nil || !strings.Contains(err.Error(), "ollama status 400") {
			t.Fatalf("expected status error, got %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer server.Close()

		provider := NewOllamaProvider(server.URL, "qwen")
		_, err := provider.Generate(context.Background(), GenerateRequest{})
		if err == nil || !strings.Contains(err.Error(), "parse response") {
			t.Fatalf("expected parse response error, got %v", err)
		}
	})

	t.Run("request failure", func(t *testing.T) {
		provider := NewOllamaProvider("http://127.0.0.1:1", "qwen")
		provider.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("dial failed")
		})}
		_, err := provider.Generate(context.Background(), GenerateRequest{})
		if err == nil || !strings.Contains(err.Error(), "ollama request") {
			t.Fatalf("expected request error, got %v", err)
		}
	})
}

func TestOpenAIGenerate_SuccessAndErrorBranches(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var authHeader string
		var captured openAIRequest

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader = r.Header.Get("Authorization")
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if err := json.Unmarshal(body, &captured); err != nil {
				t.Fatalf("unmarshal request: %v", err)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}],"usage":{"total_tokens":42}}`))
		}))
		defer server.Close()

		provider := &OpenAIProvider{
			apiKey:  "test-key",
			model:   "gpt-test",
			baseURL: server.URL,
			client:  server.Client(),
		}
		resp, err := provider.Generate(context.Background(), GenerateRequest{
			System: "system",
			Messages: []Message{
				{Role: "user", Content: "generate"},
			},
			SchemaProperties: map[string]any{
				"ok": map[string]any{"type": "boolean"},
			},
			SchemaRequired: []string{"ok"},
			Temperature:    0.2,
			MaxTokens:      256,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if authHeader != "Bearer test-key" {
			t.Fatalf("unexpected auth header %q", authHeader)
		}
		if captured.Model != "gpt-test" {
			t.Fatalf("unexpected model %q", captured.Model)
		}
		if captured.ResponseFormat == nil || captured.ResponseFormat.JSONSchema == nil {
			t.Fatal("expected json schema response format")
		}
		if resp.Tokens != 42 {
			t.Fatalf("unexpected token count %d", resp.Tokens)
		}
		if string(resp.RawJSON) != `{"ok":true}` {
			t.Fatalf("unexpected raw json %q", string(resp.RawJSON))
		}
	})

	t.Run("status error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "boom", http.StatusUnauthorized)
		}))
		defer server.Close()

		provider := &OpenAIProvider{apiKey: "test-key", model: "gpt-test", baseURL: server.URL, client: server.Client()}
		_, err := provider.Generate(context.Background(), GenerateRequest{})
		if err == nil || !strings.Contains(err.Error(), "openai api error (status 401)") {
			t.Fatalf("expected status error, got %v", err)
		}
	})

	t.Run("api error payload", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"error":{"message":"rate limited","type":"rate_limit"}}`))
		}))
		defer server.Close()

		provider := &OpenAIProvider{apiKey: "test-key", model: "gpt-test", baseURL: server.URL, client: server.Client()}
		_, err := provider.Generate(context.Background(), GenerateRequest{})
		if err == nil || !strings.Contains(err.Error(), "openai error: rate limited (rate_limit)") {
			t.Fatalf("expected api error, got %v", err)
		}
	})

	t.Run("no choices", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"choices":[],"usage":{"total_tokens":1}}`))
		}))
		defer server.Close()

		provider := &OpenAIProvider{apiKey: "test-key", model: "gpt-test", baseURL: server.URL, client: server.Client()}
		_, err := provider.Generate(context.Background(), GenerateRequest{})
		if err == nil || !strings.Contains(err.Error(), "no choices in response") {
			t.Fatalf("expected no choices error, got %v", err)
		}
	})

	t.Run("request failure", func(t *testing.T) {
		provider := &OpenAIProvider{
			apiKey:  "test-key",
			model:   "gpt-test",
			baseURL: "http://127.0.0.1:1",
			client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("transport down")
			})},
		}
		_, err := provider.Generate(context.Background(), GenerateRequest{})
		if err == nil || !strings.Contains(err.Error(), "http request") {
			t.Fatalf("expected request failure, got %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer server.Close()

		provider := &OpenAIProvider{apiKey: "test-key", model: "gpt-test", baseURL: server.URL, client: server.Client()}
		_, err := provider.Generate(context.Background(), GenerateRequest{})
		if err == nil || !strings.Contains(err.Error(), "unmarshal response") {
			t.Fatalf("expected unmarshal error, got %v", err)
		}
	})
}

func TestGeminiGenerate_SuccessAndErrorBranches(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var apiKeyHeader string
		var captured geminiRequest

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKeyHeader = r.Header.Get("x-goog-api-key")
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if err := json.Unmarshal(body, &captured); err != nil {
				t.Fatalf("unmarshal request: %v", err)
			}
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"ok\":true}"}]}}],"usageMetadata":{"totalTokenCount":17}}`))
		}))
		defer server.Close()

		provider := &GeminiProvider{
			apiKey:  "gem-key",
			model:   "gemini-test",
			baseURL: server.URL,
			client:  server.Client(),
		}
		resp, err := provider.Generate(context.Background(), GenerateRequest{
			System: "rules",
			Messages: []Message{
				{Role: "user", Content: "generate"},
				{Role: "assistant", Content: "draft"},
			},
			SchemaProperties: map[string]any{
				"ok": map[string]any{"type": "boolean"},
			},
			SchemaRequired: []string{"ok"},
			Temperature:    0.1,
			MaxTokens:      99,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if apiKeyHeader != "gem-key" {
			t.Fatalf("unexpected api key header %q", apiKeyHeader)
		}
		if captured.SystemInstruction == nil || len(captured.SystemInstruction.Parts) != 1 {
			t.Fatal("expected system instruction")
		}
		if len(captured.Contents) != 2 || captured.Contents[1].Role != "model" {
			t.Fatalf("expected assistant role remapped to model, got %#v", captured.Contents)
		}
		if captured.GenerationConfig.ResponseSchema == nil {
			t.Fatal("expected response schema")
		}
		if resp.Tokens != 17 {
			t.Fatalf("unexpected token count %d", resp.Tokens)
		}
	})

	t.Run("status error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "denied", http.StatusForbidden)
		}))
		defer server.Close()

		provider := &GeminiProvider{apiKey: "gem-key", model: "gemini-test", baseURL: server.URL, client: server.Client()}
		_, err := provider.Generate(context.Background(), GenerateRequest{})
		if err == nil || !strings.Contains(err.Error(), "gemini api error (status 403)") {
			t.Fatalf("expected status error, got %v", err)
		}
	})

	t.Run("api error payload", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"error":{"message":"quota","code":429}}`))
		}))
		defer server.Close()

		provider := &GeminiProvider{apiKey: "gem-key", model: "gemini-test", baseURL: server.URL, client: server.Client()}
		_, err := provider.Generate(context.Background(), GenerateRequest{})
		if err == nil || !strings.Contains(err.Error(), "gemini error (code 429): quota") {
			t.Fatalf("expected api error, got %v", err)
		}
	})

	t.Run("no content", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"candidates":[],"usageMetadata":{"totalTokenCount":1}}`))
		}))
		defer server.Close()

		provider := &GeminiProvider{apiKey: "gem-key", model: "gemini-test", baseURL: server.URL, client: server.Client()}
		_, err := provider.Generate(context.Background(), GenerateRequest{})
		if err == nil || !strings.Contains(err.Error(), "no content in response") {
			t.Fatalf("expected no content error, got %v", err)
		}
	})

	t.Run("request failure", func(t *testing.T) {
		provider := &GeminiProvider{
			apiKey:  "gem-key",
			model:   "gemini-test",
			baseURL: "http://127.0.0.1:1",
			client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("transport down")
			})},
		}
		_, err := provider.Generate(context.Background(), GenerateRequest{})
		if err == nil || !strings.Contains(err.Error(), "http request") {
			t.Fatalf("expected request failure, got %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer server.Close()

		provider := &GeminiProvider{apiKey: "gem-key", model: "gemini-test", baseURL: server.URL, client: server.Client()}
		_, err := provider.Generate(context.Background(), GenerateRequest{})
		if err == nil || !strings.Contains(err.Error(), "unmarshal response") {
			t.Fatalf("expected unmarshal error, got %v", err)
		}
	})
}

func TestClaudeExtractors(t *testing.T) {
	resp := &anthropic.Message{
		Content: []anthropic.ContentBlockUnion{
			{Type: "text", Text: "hello"},
			{Type: "tool_use", Input: []byte(`{"pattern":"ok"}`)},
		},
	}

	if got := extractToolUseInput(resp); got != `{"pattern":"ok"}` {
		t.Fatalf("unexpected tool input %q", got)
	}
	if got := extractTextContent(resp); got != "hello" {
		t.Fatalf("unexpected text content %q", got)
	}
	if got := extractToolUseInput(&anthropic.Message{}); got != "" {
		t.Fatalf("expected empty tool input, got %q", got)
	}
	if got := extractTextContent(&anthropic.Message{}); got != "" {
		t.Fatalf("expected empty text content, got %q", got)
	}
}

type sequenceProvider struct {
	responses []GenerateResponse
	err       error
	calls     int
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func (p *sequenceProvider) Name() string { return "sequence" }

func (p *sequenceProvider) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	if p.err != nil {
		return GenerateResponse{}, p.err
	}
	resp := p.responses[p.calls]
	p.calls++
	return resp, nil
}

func TestGenerateWithRetry_EventualSuccessAndMaxRetries(t *testing.T) {
	t.Run("eventual success after correction", func(t *testing.T) {
		p := &sequenceProvider{
			responses: []GenerateResponse{
				{RawJSON: []byte(`{"bad":true}`), Provider: "sequence", Tokens: 11},
				{RawJSON: []byte(`{"ok":true}`), Provider: "sequence", Tokens: 7},
			},
		}
		validateCalls := 0
		raw, err := GenerateWithRetry(context.Background(), p, GenerateRequest{
			OutputSchema: []byte(`{"type":"object"}`),
			Messages:     []Message{{Role: "user", Content: "generate"}},
		}, func(raw []byte) error {
			validateCalls++
			if validateCalls == 1 {
				return assertErr("density too low")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected retry error: %v", err)
		}
		if string(raw) != `{"ok":true}` {
			t.Fatalf("unexpected final raw json %q", string(raw))
		}
		if p.calls != 2 {
			t.Fatalf("expected 2 provider calls, got %d", p.calls)
		}
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		p := &sequenceProvider{
			responses: []GenerateResponse{
				{RawJSON: []byte(`{"still":"bad"}`), Provider: "sequence", Tokens: 3},
				{RawJSON: []byte(`{"still":"bad"}`), Provider: "sequence", Tokens: 5},
				{RawJSON: []byte(`{"still":"bad"}`), Provider: "sequence", Tokens: 7},
			},
		}
		_, err := GenerateWithRetry(context.Background(), p, GenerateRequest{}, func(raw []byte) error {
			return assertErr("density too high")
		})
		if err == nil || !strings.Contains(err.Error(), "max retries exceeded (tokens used: 15)") {
			t.Fatalf("expected max retries error with token total, got %v", err)
		}
	})
}

func TestAppendCorrection_ContainsSchemaAndExamples(t *testing.T) {
	msgs := appendCorrection(
		[]Message{{Role: "user", Content: "generate"}},
		[]byte(`{"type":"object"}`),
		[]byte(`{"bad":1}`),
		assertErr("section 2 (bars 5-8): chord coherence 50% < required 75% for bassline"),
		false,
	)
	if len(msgs) != 2 {
		t.Fatalf("expected appended correction message, got %d", len(msgs))
	}
	body := msgs[1].Content
	if !strings.Contains(body, "<json_schema>") || !strings.Contains(body, "<correction_example>") {
		t.Fatalf("expected schema and example blocks, got %q", body)
	}
	if !strings.Contains(body, "failed musical validation") {
		t.Fatalf("expected musical validation instruction, got %q", body)
	}
}

func TestClaudeGenerate_ToolUseAndTextFallback(t *testing.T) {
	t.Run("tool use response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id":"msg_123",
				"type":"message",
				"role":"assistant",
				"model":"claude-test",
				"content":[{"type":"tool_use","id":"toolu_123","name":"generate_pattern","input":{"pattern":"ok"}}],
				"stop_reason":"tool_use",
				"stop_sequence":"",
				"usage":{"input_tokens":10,"output_tokens":5}
			}`))
		}))
		defer server.Close()

		client := anthropic.NewClient(option.WithAPIKey("x"), option.WithBaseURL(server.URL))
		provider := &ClaudeProvider{client: &client, model: "claude-test"}
		resp, err := provider.Generate(context.Background(), GenerateRequest{
			System: "rules",
			Messages: []Message{
				{Role: "user", Content: "generate"},
			},
			OutputSchema: []byte(`{"pattern":"ok"}`),
		})
		if err != nil {
			t.Fatalf("unexpected claude error: %v", err)
		}
		if string(resp.RawJSON) != `{"pattern":"ok"}` {
			t.Fatalf("unexpected tool use payload %q", string(resp.RawJSON))
		}
		if resp.Tokens != 15 {
			t.Fatalf("unexpected token count %d", resp.Tokens)
		}
	})

	t.Run("text fallback", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id":"msg_124",
				"type":"message",
				"role":"assistant",
				"model":"claude-test",
				"content":[{"type":"text","text":"{\"pattern\":\"text\"}"}],
				"stop_reason":"end_turn",
				"stop_sequence":"",
				"usage":{"input_tokens":1,"output_tokens":2}
			}`))
		}))
		defer server.Close()

		client := anthropic.NewClient(option.WithAPIKey("x"), option.WithBaseURL(server.URL))
		provider := &ClaudeProvider{client: &client, model: "claude-test"}
		resp, err := provider.Generate(context.Background(), GenerateRequest{
			Messages: []Message{
				{Role: "assistant", Content: "partial"},
				{Role: "user", Content: "generate"},
			},
		})
		if err != nil {
			t.Fatalf("unexpected claude error: %v", err)
		}
		if string(resp.RawJSON) != `{"pattern":"text"}` {
			t.Fatalf("unexpected text fallback payload %q", string(resp.RawJSON))
		}
	})
}
