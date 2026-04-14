package functiontool

import (
	"encoding/json"
	"fmt"
)

// mapToStruct converts a map[string]any to a typed struct.
// This uses JSON marshaling/unmarshaling to handle type conversion properly.
func mapToStruct(m map[string]any, target any) error {
	if m == nil {
		return nil
	}

	// Marshal map to JSON
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}

	// Unmarshal JSON to target struct
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal args: %w", err)
	}

	return nil
}
