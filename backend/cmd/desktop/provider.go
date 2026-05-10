package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/Andrea-Cavallo/cadenza/internal/llm"
	"github.com/Andrea-Cavallo/cadenza/internal/models"
)

func buildProvider(noLLM bool, providerName, model string) (llm.Provider, error) {
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	if noLLM || providerName == "" || providerName == "offline" {
		return &llm.MockProvider{}, nil
	}
	if model == "" {
		model = models.DefaultModel(providerName)
	}

	switch providerName {
	case "claude":
		return llm.NewClaudeProvider(model)
	case "ollama":
		// Local models are slow; 5-minute timeout per request, run sequentially via mg.Sequential.
		return llm.NewOllamaProviderWithTimeout("http://localhost:11434", model, 5*time.Minute), nil
	case "openai":
		return llm.NewOpenAIProvider(model)
	case "gemini":
		return llm.NewGeminiProvider(model)
	default:
		return nil, fmt.Errorf("unknown provider %q (supported: claude, ollama, openai, gemini, offline)", providerName)
	}
}
