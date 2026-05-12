package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// AppConfig holds the full application configuration.
type AppConfig struct {
	App      AppSection      `mapstructure:"app"`
	Audio    AudioSection    `mapstructure:"audio"`
	LLM      LLMSection      `mapstructure:"llm"`
	Renderer RendererSection `mapstructure:"renderer"`
	Output   OutputSection   `mapstructure:"output"`
	Logging  LoggingSection  `mapstructure:"logging"`
	Cache    CacheSection    `mapstructure:"cache"`
	Session  SessionSection  `mapstructure:"session"`
}

// AppSection holds general application settings.
type AppSection struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Env     string `mapstructure:"env"` // development, production
}

// AudioSection holds musical defaults.
type AudioSection struct {
	BPM          float64 `mapstructure:"bpm"`
	Key          string  `mapstructure:"key"`
	Bars         int     `mapstructure:"bars"`
	Variations   int     `mapstructure:"variations"`
	Groove       string  `mapstructure:"groove"`
	TicksPerBeat int     `mapstructure:"ticks_per_beat"`
	MaxVelocity  int     `mapstructure:"max_velocity"`
}

// LLMSection holds LLM provider configuration.
type LLMSection struct {
	Provider    string  `mapstructure:"provider"`
	Model       string  `mapstructure:"model"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	MaxRetries  int     `mapstructure:"max_retries"`
	Timeout     int     `mapstructure:"timeout_seconds"`
	OllamaURL   string  `mapstructure:"ollama_url"`
}

// RendererSection holds rendering defaults.
type RendererSection struct {
	BassRange    [2]int `mapstructure:"bass_range"`
	ArpRange     [2]int `mapstructure:"arp_range"`
	MelodyRange  [2]int `mapstructure:"melody_range"`
	GhostVelMin  int    `mapstructure:"ghost_velocity_min"`
	GhostVelMax  int    `mapstructure:"ghost_velocity_max"`
	PortamentoCC int    `mapstructure:"portamento_cc"`
}

// OutputSection holds output path configuration.
type OutputSection struct {
	Dir        string `mapstructure:"dir"`
	FilePrefix string `mapstructure:"file_prefix"`
	MIDIType   int    `mapstructure:"midi_type"` // 0 or 1
}

// LoggingSection holds logging configuration.
type LoggingSection struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // text, json
	File   string `mapstructure:"file"`
}

// CacheSection holds cache configuration.
type CacheSection struct {
	Enabled bool   `mapstructure:"enabled"`
	TTLDays int    `mapstructure:"ttl_days"`
	Dir     string `mapstructure:"dir"`
}

// SessionSection parametrizza la persistenza delle sessioni su disco.
type SessionSection struct {
	Dir                string `mapstructure:"dir"`
	MaxSizeMB          int    `mapstructure:"max_size_mb"`
	CheckpointInterval int    `mapstructure:"checkpoint_interval"`
	MaxSessions        int    `mapstructure:"max_sessions"`
}

// Load reads configuration from file, env vars, and applies defaults.
func Load() (*AppConfig, error) {
	v := viper.New()

	setDefaults(v)

	v.SetConfigName("cadenza")
	v.SetConfigType("yaml")

	// Search paths
	v.AddConfigPath(".")
	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(filepath.Join(home, ".cadenza"))
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		v.AddConfigPath(filepath.Join(xdg, "cadenza"))
	}

	// Environment variables: CADENZA_AUDIO_BPM, CADENZA_LLM_PROVIDER, etc.
	v.SetEnvPrefix("CADENZA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Map common env vars to config keys
	_ = v.BindEnv("llm.provider", "CADENZA_LLM_PROVIDER")
	_ = v.BindEnv("llm.model", "CADENZA_LLM_MODEL")
	_ = v.BindEnv("logging.level", "CADENZA_LOG_LEVEL")
	_ = v.BindEnv("logging.format", "CADENZA_LOG_FORMAT")
	_ = v.BindEnv("app.env", "CADENZA_ENV")
	_ = v.BindEnv("output.dir", "CADENZA_OUTPUT_DIR")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// App
	v.SetDefault("app.name", "cadenza")
	v.SetDefault("app.version", "dev")
	v.SetDefault("app.env", "development")

	// Audio
	v.SetDefault("audio.bpm", 122.0)
	v.SetDefault("audio.key", "Am")
	v.SetDefault("audio.bars", 16)
	v.SetDefault("audio.variations", 1)
	v.SetDefault("audio.groove", "straight")
	v.SetDefault("audio.ticks_per_beat", 480)
	v.SetDefault("audio.max_velocity", 120)

	// LLM
	v.SetDefault("llm.provider", "claude")
	v.SetDefault("llm.model", "claude-opus-4-7")
	v.SetDefault("llm.temperature", 0.3)
	v.SetDefault("llm.max_tokens", 4096)
	v.SetDefault("llm.max_retries", 3)
	v.SetDefault("llm.timeout_seconds", 30)
	v.SetDefault("llm.ollama_url", "http://localhost:11434")

	// Renderer
	v.SetDefault("renderer.bass_range", []int{33, 55})
	v.SetDefault("renderer.arp_range", []int{48, 84})
	v.SetDefault("renderer.melody_range", []int{60, 96})
	v.SetDefault("renderer.ghost_velocity_min", 35)
	v.SetDefault("renderer.ghost_velocity_max", 55)
	v.SetDefault("renderer.portamento_cc", 5)

	// Output
	v.SetDefault("output.dir", "output")
	v.SetDefault("output.file_prefix", "output")
	v.SetDefault("output.midi_type", 0)

	// Logging
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.file", "cadenza.log")

	// Cache
	v.SetDefault("cache.enabled", true)
	v.SetDefault("cache.ttl_days", 30)
	v.SetDefault("cache.dir", ".cache")

	// Session
	home, _ := os.UserHomeDir()
	v.SetDefault("session.dir", filepath.Join(home, ".cadenza", "sessions"))
	v.SetDefault("session.max_size_mb", 512)
	v.SetDefault("session.checkpoint_interval", 3)
	v.SetDefault("session.max_sessions", 20)
}

func (c *AppConfig) Validate() error {
	var errs []string

	if c.Audio.MaxVelocity < 1 || c.Audio.MaxVelocity > 127 {
		errs = append(errs, fmt.Sprintf("audio.max_velocity must be 1-127, got %d", c.Audio.MaxVelocity))
	}
	if c.Audio.BPM < 60 || c.Audio.BPM > 200 {
		errs = append(errs, fmt.Sprintf("audio.bpm must be 60-200, got %.1f", c.Audio.BPM))
	}
	if c.Audio.Bars < 1 || c.Audio.Bars > 128 {
		errs = append(errs, fmt.Sprintf("audio.bars must be 1-128, got %d", c.Audio.Bars))
	}
	if c.Cache.TTLDays < 0 {
		errs = append(errs, fmt.Sprintf("cache.ttl_days must be >= 0, got %d", c.Cache.TTLDays))
	}

	validEnvs := map[string]bool{"development": true, "production": true, "test": true}
	if !validEnvs[c.App.Env] {
		errs = append(errs, fmt.Sprintf("app.env must be development/production/test, got %q", c.App.Env))
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		errs = append(errs, fmt.Sprintf("logging.level must be debug/info/warn/error, got %q", c.Logging.Level))
	}

	validFormats := map[string]bool{"text": true, "json": true}
	if !validFormats[c.Logging.Format] {
		errs = append(errs, fmt.Sprintf("logging.format must be text/json, got %q", c.Logging.Format))
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n- %s", strings.Join(errs, "\n- "))
	}
	return nil
}
