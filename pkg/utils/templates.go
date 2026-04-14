package utils

import (
	"encoding/json"
	"text/template"
	"time"
)

// TemplateFuncs returns common template helper functions used across Hector.
// These are useful for payload transformation in webhooks and notifications.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		// toJson converts a value to a compact JSON string.
		"toJson": func(v any) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		// toJsonPretty converts a value to a pretty-printed JSON string.
		"toJsonPretty": func(v any) string {
			b, _ := json.MarshalIndent(v, "", "  ")
			return string(b)
		},
		// now returns the current time in RFC3339 format.
		"now": func() string {
			return time.Now().Format(time.RFC3339)
		},
		// default returns the default value if the value is nil or empty.
		"default": func(def, val any) any {
			if val == nil || val == "" {
				return def
			}
			return val
		},
	}
}
