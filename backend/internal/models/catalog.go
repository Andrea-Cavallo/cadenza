package models

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed models.yaml
var embeddedFS embed.FS

// ModelEntry describes one model available for a provider.
type ModelEntry struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// ProviderConfig holds the default model and full list for one provider.
type ProviderConfig struct {
	Default string       `yaml:"default"`
	Models  []ModelEntry `yaml:"models"`
}

// Catalog is the full model registry.
type Catalog struct {
	Providers map[string]ProviderConfig `yaml:"providers"`
}

var global *Catalog

// Load initialises the global catalog. Call once at startup.
// It checks (in order): localPath arg → ~/.cadenza/models.yaml → ./models.yaml → embedded default.
func Load(localPath string) {
	candidates := []string{localPath}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".cadenza", "models.yaml"))
	}
	candidates = append(candidates, "models.yaml")

	for _, path := range candidates {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		c, err := parse(data)
		if err != nil {
			slog.Warn("models catalog parse error, falling back to embedded", "path", path, "error", err)
			break
		}
		slog.Info("models catalog loaded from file", "path", path)
		global = c
		return
	}

	data, err := embeddedFS.ReadFile("models.yaml")
	if err != nil {
		panic(fmt.Sprintf("embedded models.yaml missing: %v", err))
	}
	c, err := parse(data)
	if err != nil {
		panic(fmt.Sprintf("embedded models.yaml invalid: %v", err))
	}
	global = c
}

// DefaultModel returns the default model ID for the given provider.
// Falls back to the empty string when the provider is not in the catalog.
func DefaultModel(provider string) string {
	ensureLoaded()
	if pc, ok := global.Providers[provider]; ok {
		return pc.Default
	}
	return ""
}

// List returns all ModelEntry values for the given provider.
func List(provider string) []ModelEntry {
	ensureLoaded()
	if pc, ok := global.Providers[provider]; ok {
		return pc.Models
	}
	return nil
}

// Providers returns all provider names defined in the catalog.
func Providers() []string {
	ensureLoaded()
	keys := make([]string, 0, len(global.Providers))
	for k := range global.Providers {
		keys = append(keys, k)
	}
	return keys
}

func parse(data []byte) (*Catalog, error) {
	var c Catalog
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func ensureLoaded() {
	if global == nil {
		Load("")
	}
}
