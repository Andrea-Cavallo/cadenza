package schema

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

// GenerateJSONSchema generates a JSON Schema from the PatternSpec struct using invopop/jsonschema.
// This replaces manual schema inference from example JSON, providing a complete and accurate schema
// that includes all fields, proper types, and validation constraints.
func GenerateJSONSchema() (properties map[string]any, required []string, err error) {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true, // inline all definitions
	}

	schemaObj := reflector.Reflect(&PatternSpec{})

	// Marshal and unmarshal to get a clean map[string]any representation
	schemaBytes, err := json.Marshal(schemaObj)
	if err != nil {
		return nil, nil, err
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return nil, nil, err
	}

	// Extract properties and required fields from the generated schema
	var ok bool
	properties, ok = schemaMap["properties"].(map[string]any)
	if !ok {
		properties = make(map[string]any)
	}

	if reqList, ok := schemaMap["required"].([]any); ok {
		for _, r := range reqList {
			if s, ok := r.(string); ok {
				required = append(required, s)
			}
		}
	}

	return properties, required, nil
}
