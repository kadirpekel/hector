package builder

import (
	"github.com/verikod/hector/pkg/tool"
	"github.com/verikod/hector/pkg/tool/functiontool"
)

// FunctionTool creates a callable tool from a typed Go function.
//
// This is a convenience wrapper around functiontool.New for ergonomic use.
// The function signature must be:
//
//	func(tool.Context, Args) (map[string]any, error)
//
// Where Args is a struct with json and jsonschema tags defining the parameters.
//
// Example:
//
//	type GetWeatherArgs struct {
//	    City string `json:"city" jsonschema:"required,description=City name"`
//	}
//
//	tool, err := builder.FunctionTool(
//	    "get_weather",
//	    "Get current weather for a city",
//	    func(ctx tool.Context, args GetWeatherArgs) (map[string]any, error) {
//	        return map[string]any{"temp": 22}, nil
//	    },
//	)
func FunctionTool[Args any](
	name string,
	description string,
	fn func(tool.Context, Args) (map[string]any, error),
) (tool.CallableTool, error) {
	return functiontool.New(functiontool.Config{
		Name:        name,
		Description: description,
	}, fn)
}

// MustFunctionTool creates a callable tool or panics on error.
//
// Use this only when you're certain the configuration is valid.
func MustFunctionTool[Args any](
	name string,
	description string,
	fn func(tool.Context, Args) (map[string]any, error),
) tool.CallableTool {
	t, err := FunctionTool(name, description, fn)
	if err != nil {
		panic("failed to create function tool: " + err.Error())
	}
	return t
}

// boolPtr returns a pointer to the given bool value.
//
//nolint:unused // Reserved for future use
func boolPtr(b bool) *bool {
	return &b
}
