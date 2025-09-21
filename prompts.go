package hector

import (
	"fmt"
	"strings"
	"text/template"
)

// ============================================================================
// PROMPT GENERATION SYSTEM
// ============================================================================

// PromptData holds data for prompt template rendering
type PromptData struct {
	Query        string
	Context      []string
	ContextCount int
	ModelName    string
	Instruction  string
}

// DefaultAgentTemplate is the enhanced agent prompt template
const DefaultAgentTemplate = `{{if .Instruction}}{{.Instruction}}{{else}}You are a helpful assistant with access to knowledge sources.{{end}} Use the following context to answer the user's question.

{{if .Context}}
{{range $i, $ctx := .Context}}{{printf "%s" $ctx}}
{{end}}
{{end}}
Question: {{.Query}}

Answer:`

// BuildPrompt builds a prompt for the given query and context
// Uses customTemplate if provided, otherwise uses DefaultAgentTemplate
func BuildPrompt(query string, context []string, modelName string, instruction string, customTemplate string) (string, error) {
	var templateStr string

	if customTemplate != "" {
		templateStr = customTemplate
	} else {
		templateStr = DefaultAgentTemplate
	}

	// Parse template
	tmpl, err := template.New("prompt").Funcs(template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"title": strings.Title,
	}).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Prepare data
	data := PromptData{
		Query:        query,
		Context:      context,
		ContextCount: len(context),
		ModelName:    modelName,
		Instruction:  instruction,
	}

	// Render template
	var buf strings.Builder
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}
