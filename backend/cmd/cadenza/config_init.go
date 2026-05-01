package main

import (
	"fmt"
	"os"
	"strings"
)

const defaultConfigPath = "cadenza.yaml"

func handleConfigCommand(args []string) bool {
	handled, force, err := parseConfigInitArgs(args)
	if !handled {
		return false
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := runConfigInit(defaultConfigPath, force); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return true
}

func parseConfigInitArgs(args []string) (handled bool, force bool, err error) {
	if len(args) < 2 || args[0] != "config" {
		return false, false, nil
	}
	if args[1] != "init" {
		return true, false, fmt.Errorf("usage: cadenza config init [--force]")
	}
	for _, arg := range args[2:] {
		if arg == "--force" {
			force = true
			continue
		}
		return true, false, fmt.Errorf("unknown config init option %q", arg)
	}
	return true, force, nil
}

func runConfigInit(path string, force bool) error {
	if path == "" {
		path = defaultConfigPath
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite)", path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("check existing config: %w", err)
		}
	}
	if err := os.WriteFile(path, []byte(starterConfigYAML()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("Created %s\n", path)
	return nil
}

func starterConfigYAML() string {
	return strings.TrimLeft(`# cadenza.yaml
# Starter configuration for Cadenza.
# CLI flags always override these values.

app:
  env: development # development | production | test

audio:
  bpm: 122
  key: Am
  bars: 16 # 16 | 32 | 64 | 128
  variations: 1
  groove: straight # straight | mpc60 | linndrum | humanize
  ticks_per_beat: 480
  max_velocity: 120

llm:
  provider: claude # claude | ollama | openai | gemini
  model: claude-opus-4-7
  temperature: 0.3
  max_tokens: 4096
  max_retries: 3
  timeout_seconds: 30
  ollama_url: http://localhost:11434

renderer:
  bass_range: [33, 55]
  arp_range: [48, 84]
  melody_range: [60, 96]
  ghost_velocity_min: 35
  ghost_velocity_max: 55
  portamento_cc: 5

output:
  dir: output
  file_prefix: output
  midi_type: 0

logging:
  level: info # debug | info | warn | error
  format: text # text | json
  file: cadenza.log

cache:
  enabled: true
  ttl_days: 30
  dir: .cache
`, "\n")
}
