package main

import (
	"fmt"
	"log/slog"

	"github.com/Andrea-Cavallo/cadenza/internal/config"
	"github.com/Andrea-Cavallo/cadenza/internal/llm"
	"github.com/Andrea-Cavallo/cadenza/internal/models"
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

// resolveDefaultModel returns the default model for a provider from the catalog,
// falling back to the app config value for claude (which may be set by the user).
func resolveDefaultModel(provider string, appCfg *config.AppConfig) string {
	if provider == "claude" && appCfg.LLM.Model != "" {
		return appCfg.LLM.Model
	}
	if def := models.DefaultModel(provider); def != "" {
		return def
	}
	return appCfg.LLM.Model
}
