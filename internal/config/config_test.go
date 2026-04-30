package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Load from a temp dir with no config file — should fall back to defaults
	dir := t.TempDir()
	t.Chdir(dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Audio.BPM != 122 {
		t.Errorf("expected default BPM 122, got %v", cfg.Audio.BPM)
	}
	if cfg.LLM.Provider != "claude" {
		t.Errorf("expected default provider claude, got %q", cfg.LLM.Provider)
	}
	if cfg.Audio.Bars != 16 {
		t.Errorf("expected default bars 16, got %d", cfg.Audio.Bars)
	}
	if cfg.Cache.TTLDays != 30 {
		t.Errorf("expected default ttl_days 30, got %d", cfg.Cache.TTLDays)
	}
}

func TestLoadFromYAML(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `
audio:
  bpm: 130
  key: "Dm"
  bars: 32
llm:
  provider: "ollama"
  model: "qwen2.5:7b"
`
	if err := os.WriteFile(filepath.Join(dir, "cadenza.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Chdir(dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Audio.BPM != 130 {
		t.Errorf("expected BPM 130, got %v", cfg.Audio.BPM)
	}
	if cfg.Audio.Bars != 32 {
		t.Errorf("expected bars 32, got %d", cfg.Audio.Bars)
	}
	if cfg.LLM.Provider != "ollama" {
		t.Errorf("expected provider ollama, got %q", cfg.LLM.Provider)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	t.Setenv("CADENZA_LLM_PROVIDER", "openai")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.LLM.Provider != "openai" {
		t.Errorf("expected env override provider openai, got %q", cfg.LLM.Provider)
	}
}

func TestValidateValid(t *testing.T) {
	cfg := &AppConfig{
		App:     AppSection{Env: "development"},
		Audio:   AudioSection{BPM: 122, MaxVelocity: 120, Bars: 16},
		Logging: LoggingSection{Level: "info", Format: "text"},
		Cache:   CacheSection{TTLDays: 30},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got: %v", err)
	}
}

func TestValidateBPMOutOfRange(t *testing.T) {
	cfg := &AppConfig{
		App:     AppSection{Env: "development"},
		Audio:   AudioSection{BPM: 250, MaxVelocity: 120, Bars: 16},
		Logging: LoggingSection{Level: "info", Format: "text"},
		Cache:   CacheSection{TTLDays: 30},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for BPM 250")
	}
}

func TestValidateMaxVelocity(t *testing.T) {
	cfg := &AppConfig{
		App:     AppSection{Env: "development"},
		Audio:   AudioSection{BPM: 122, MaxVelocity: 200, Bars: 16},
		Logging: LoggingSection{Level: "info", Format: "text"},
		Cache:   CacheSection{TTLDays: 30},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for max_velocity 200")
	}
}

func TestValidateInvalidEnv(t *testing.T) {
	cfg := &AppConfig{
		App:     AppSection{Env: "staging"},
		Audio:   AudioSection{BPM: 122, MaxVelocity: 120, Bars: 16},
		Logging: LoggingSection{Level: "info", Format: "text"},
		Cache:   CacheSection{TTLDays: 30},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid env")
	}
}

func TestValidateInvalidLogLevel(t *testing.T) {
	cfg := &AppConfig{
		App:     AppSection{Env: "development"},
		Audio:   AudioSection{BPM: 122, MaxVelocity: 120, Bars: 16},
		Logging: LoggingSection{Level: "trace", Format: "text"},
		Cache:   CacheSection{TTLDays: 30},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid log level")
	}
}

func TestValidateNegativeTTL(t *testing.T) {
	cfg := &AppConfig{
		App:     AppSection{Env: "development"},
		Audio:   AudioSection{BPM: 122, MaxVelocity: 120, Bars: 16},
		Logging: LoggingSection{Level: "info", Format: "text"},
		Cache:   CacheSection{TTLDays: -1},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for negative TTL")
	}
}

func TestValidateMultipleErrors(t *testing.T) {
	cfg := &AppConfig{
		App:     AppSection{Env: "invalid"},
		Audio:   AudioSection{BPM: 300, MaxVelocity: 200, Bars: 0},
		Logging: LoggingSection{Level: "invalid", Format: "xml"},
		Cache:   CacheSection{TTLDays: -5},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected multiple validation errors")
	}
}
