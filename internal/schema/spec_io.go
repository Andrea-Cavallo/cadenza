package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DumpToYAML writes a PatternSpec to a YAML file
func DumpToYAML(spec *PatternSpec, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	data, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// LoadFromYAML reads a PatternSpec from a YAML file
func LoadFromYAML(inputPath string) (*PatternSpec, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var spec PatternSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}

	return &spec, nil
}

// DumpToJSON writes a PatternSpec to a JSON file
func DumpToJSON(spec *PatternSpec, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// LoadFromJSON reads a PatternSpec from a JSON file
func LoadFromJSON(inputPath string) (*PatternSpec, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var spec PatternSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	return &spec, nil
}
