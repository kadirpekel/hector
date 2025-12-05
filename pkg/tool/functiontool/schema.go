// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel
//
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package functiontool

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
)

// generateSchema creates a JSON schema from a Go type using struct tags.
//
// Supported tags:
//   - json:"name" - Parameter name
//   - json:",omitempty" - Optional parameter
//   - jsonschema:"required" - Explicitly mark as required
//   - jsonschema:"description=..." - Parameter description
//   - jsonschema:"default=..." - Default value
//   - jsonschema:"enum=val1|val2" - Allowed values
//   - jsonschema:"minimum=N,maximum=M" - Numeric constraints
//
// Example:
//
//	type Args struct {
//	    Query string `json:"query" jsonschema:"required,description=Search query"`
//	    Limit int    `json:"limit,omitempty" jsonschema:"description=Max results,default=10,minimum=1,maximum=100"`
//	}
func generateSchema[T any]() (map[string]any, error) {
	// Create reflector with ADK-Go compatible settings
	reflector := &jsonschema.Reflector{
		// Use jsonschema tags to determine required fields
		RequiredFromJSONSchemaTags: true,

		// Don't add $ref for definitions (inline everything)
		ExpandedStruct: true,

		// Don't add $schema and $id
		DoNotReference: true,
	}

	// Generate schema for the type
	schema := reflector.Reflect(new(T))

	// Convert to map[string]any for LLM consumption
	schemaMap, err := schemaToMap(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema to map: %w", err)
	}

	// ADK-Go expects the properties directly, not wrapped in type: object
	// Extract the properties if this is an object schema
	if schemaMap["type"] == "object" {
		properties, hasProps := schemaMap["properties"]
		required := schemaMap["required"]

		result := map[string]any{
			"type":       "object",
			"properties": properties,
		}

		if hasProps && required != nil {
			result["required"] = required
		}

		// Preserve additionalProperties if set
		if addProps, ok := schemaMap["additionalProperties"]; ok {
			result["additionalProperties"] = addProps
		}

		return result, nil
	}

	return schemaMap, nil
}

// schemaToMap converts a jsonschema.Schema to map[string]any.
func schemaToMap(schema *jsonschema.Schema) (map[string]any, error) {
	// Marshal to JSON then unmarshal to map
	// This ensures all jsonschema types are properly converted
	data, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	// Remove $schema and $id if present (not needed for LLM tools)
	delete(result, "$schema")
	delete(result, "$id")

	return result, nil
}
