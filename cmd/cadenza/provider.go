package main

import (
	"fmt"
	"log/slog"

	"github.com/Andrea-Cavallo/cadenza/internal/config"
	"github.com/Andrea-Cavallo/cadenza/internal/llm"
)

func buildProvider(noLLM bool, providerName, model, ollamaURL string) (llm.Provider, error) {
	if noLLM {
		slog.Info("using offline mode (no LLM)")
		return &llm.MockProvider{}, nil
	}
	switch providerName {
	case "claude":
		return llm.NewClaudeProvider(model)
	case "ollama":
		if ollamaURL == "" {
			ollamaURL = "http://localhost:11434"
		}
		return llm.NewOllamaProvider(ollamaURL, model), nil
	case "openai":
		return llm.NewOpenAIProvider(model)
	case "gemini":
		return llm.NewGeminiProvider(model)
	default:
		return nil, fmt.Errorf("unknown provider %q (supported: claude, ollama, openai, gemini)", providerName)
	}
}

func resolveDefaultModel(provider string, appCfg *config.AppConfig) string {
	switch provider {
	case "claude":
		return appCfg.LLM.Model
	case "ollama":
		return "qwen2.5:7b"
	case "openai":
		return "gpt-4o"
	case "gemini":
		return "gemini-2.0-flash-exp"
	default:
		return appCfg.LLM.Model
	}
}
