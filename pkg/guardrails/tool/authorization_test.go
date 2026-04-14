package tool_test

import (
	"context"
	"testing"

	"github.com/verikod/hector/pkg/guardrails"
	"github.com/verikod/hector/pkg/guardrails/tool"
)

// =============================================================================
// Authorizer Tests
// =============================================================================

func TestAuthorizer_Name(t *testing.T) {
	authorizer := tool.NewAuthorizer()
	if name := authorizer.Name(); name != "tool_authorizer" {
		t.Errorf("Name() = %q, want %q", name, "tool_authorizer")
	}
}

func TestAuthorizer_BlockList(t *testing.T) {
	ctx := context.Background()
	authorizer := tool.NewAuthorizer().
		Block("bash", "web_request", "execute_code")

	tests := []struct {
		name      string
		toolName  string
		wantBlock bool
	}{
		{"blocked tool bash", "bash", true},
		{"blocked tool web_request", "web_request", true},
		{"blocked tool execute_code", "execute_code", true},
		{"allowed tool search", "search", false},
		{"allowed tool read_file", "read_file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := authorizer.Check(ctx, tt.toolName, nil)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantBlock && result.Action != guardrails.ActionBlock {
				t.Errorf("expected block for %s, got %v", tt.toolName, result.Action)
			}
			if !tt.wantBlock && result.Action != guardrails.ActionAllow {
				t.Errorf("expected allow for %s, got %v", tt.toolName, result.Action)
			}
		})
	}
}

func TestAuthorizer_AllowList(t *testing.T) {
	ctx := context.Background()
	authorizer := tool.NewAuthorizer().
		AllowOnly("search", "grep_search", "read_file")

	tests := []struct {
		name      string
		toolName  string
		wantBlock bool
	}{
		{"allowed tool search", "search", false},
		{"allowed tool grep_search", "grep_search", false},
		{"allowed tool read_file", "read_file", false},
		{"not allowed bash", "bash", true},
		{"not allowed write_file", "write_file", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := authorizer.Check(ctx, tt.toolName, nil)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantBlock && result.Action != guardrails.ActionBlock {
				t.Errorf("expected block for %s, got %v", tt.toolName, result.Action)
			}
			if !tt.wantBlock && result.Action != guardrails.ActionAllow {
				t.Errorf("expected allow for %s, got %v", tt.toolName, result.Action)
			}
		})
	}
}

func TestAuthorizer_GlobPatterns(t *testing.T) {
	ctx := context.Background()
	authorizer := tool.NewAuthorizer().
		AllowOnly("read_*", "search").
		Block("*_dangerous")

	tests := []struct {
		name      string
		toolName  string
		wantBlock bool
		reason    string
	}{
		{"exact match search", "search", false, "exact match in allow list"},
		{"glob match read_file", "read_file", false, "matches read_*"},
		{"glob match read_config", "read_config", false, "matches read_*"},
		{"blocked by allow list", "write_file", true, "not in allow list"},
		{"blocked by pattern", "read_dangerous", true, "matches *_dangerous block pattern"},
		{"blocked by pattern 2", "execute_dangerous", true, "matches *_dangerous"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := authorizer.Check(ctx, tt.toolName, nil)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantBlock && result.Action != guardrails.ActionBlock {
				t.Errorf("[%s] expected block, got %v", tt.reason, result.Action)
			}
			if !tt.wantBlock && result.Action != guardrails.ActionAllow {
				t.Errorf("[%s] expected allow, got %v", tt.reason, result.Action)
			}
		})
	}
}

func TestAuthorizer_BlockTakesPrecedence(t *testing.T) {
	ctx := context.Background()
	// Tool is in allow list but also in block list - block should win
	authorizer := tool.NewAuthorizer().
		AllowOnly("dangerous_tool", "safe_tool").
		Block("dangerous_tool")

	result, _ := authorizer.Check(ctx, "dangerous_tool", nil)
	if result.Action != guardrails.ActionBlock {
		t.Error("block list should take precedence over allow list")
	}

	result, _ = authorizer.Check(ctx, "safe_tool", nil)
	if result.Action != guardrails.ActionAllow {
		t.Error("safe_tool should be allowed")
	}
}

func TestAuthorizer_CaseInsensitive(t *testing.T) {
	ctx := context.Background()
	authorizer := tool.NewAuthorizer().
		Block("BASH", "Web_Request")

	tests := []struct {
		name      string
		toolName  string
		wantBlock bool
	}{
		{"lowercase bash", "bash", true},
		{"uppercase BASH", "BASH", true},
		{"mixed case Bash", "Bash", true},
		{"lowercase web_request", "web_request", true},
		{"uppercase WEB_REQUEST", "WEB_REQUEST", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := authorizer.Check(ctx, tt.toolName, nil)
			if tt.wantBlock && result.Action != guardrails.ActionBlock {
				t.Errorf("expected block for %s", tt.toolName)
			}
		})
	}
}

func TestAuthorizer_NoRestrictions(t *testing.T) {
	ctx := context.Background()
	authorizer := tool.NewAuthorizer() // No allow or block list

	// Everything should be allowed
	for _, toolName := range []string{"bash", "search", "anything"} {
		result, _ := authorizer.Check(ctx, toolName, nil)
		if result.Action != guardrails.ActionAllow {
			t.Errorf("tool %s should be allowed with no restrictions", toolName)
		}
	}
}

func TestAuthorizer_Details(t *testing.T) {
	ctx := context.Background()
	authorizer := tool.NewAuthorizer().Block("bash")

	result, _ := authorizer.Check(ctx, "bash", nil)

	if result.Details == nil {
		t.Fatal("Details should not be nil")
	}
	if result.Details["tool"] != "bash" {
		t.Errorf("Details[tool] = %v, want bash", result.Details["tool"])
	}
	if result.Details["list"] != "blocked" {
		t.Errorf("Details[list] = %v, want blocked", result.Details["list"])
	}
}

func TestAuthorizer_ActionAndSeverity(t *testing.T) {
	ctx := context.Background()
	authorizer := tool.NewAuthorizer().
		Block("bash").
		WithAction(guardrails.ActionWarn).
		WithSeverity(guardrails.SeverityCritical)

	result, _ := authorizer.Check(ctx, "bash", nil)

	if result.Action != guardrails.ActionWarn {
		t.Errorf("Action = %v, want %v", result.Action, guardrails.ActionWarn)
	}
	if result.Severity != guardrails.SeverityCritical {
		t.Errorf("Severity = %v, want %v", result.Severity, guardrails.SeverityCritical)
	}
}

func TestAuthorizer_InterfaceCompliance(t *testing.T) {
	var _ guardrails.ToolGuardrail = tool.NewAuthorizer()
}

// =============================================================================
// ArgumentValidator Tests
// =============================================================================

func TestArgumentValidator_Name(t *testing.T) {
	validator := tool.NewArgumentValidator()
	if name := validator.Name(); name != "argument_validator" {
		t.Errorf("Name() = %q, want %q", name, "argument_validator")
	}
}

func TestArgumentValidator_RequiredArgs(t *testing.T) {
	ctx := context.Background()
	validator := tool.NewArgumentValidator().
		RequireArgs("search", "query", "max_results")

	tests := []struct {
		name      string
		toolName  string
		args      map[string]any
		wantBlock bool
		reason    string
	}{
		{
			name:      "all required present",
			toolName:  "search",
			args:      map[string]any{"query": "test", "max_results": 10},
			wantBlock: false,
		},
		{
			name:      "missing query",
			toolName:  "search",
			args:      map[string]any{"max_results": 10},
			wantBlock: true,
			reason:    "query is required",
		},
		{
			name:      "missing max_results",
			toolName:  "search",
			args:      map[string]any{"query": "test"},
			wantBlock: true,
			reason:    "max_results is required",
		},
		{
			name:      "missing all required",
			toolName:  "search",
			args:      map[string]any{},
			wantBlock: true,
			reason:    "all required args missing",
		},
		{
			name:      "different tool no requirements",
			toolName:  "other_tool",
			args:      map[string]any{},
			wantBlock: false,
			reason:    "no requirements for other_tool",
		},
		{
			name:      "extra args ok",
			toolName:  "search",
			args:      map[string]any{"query": "test", "max_results": 10, "extra": true},
			wantBlock: false,
			reason:    "extra args should not cause issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Check(ctx, tt.toolName, tt.args)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantBlock && result.Action != guardrails.ActionBlock {
				t.Errorf("[%s] expected block, got %v", tt.reason, result.Action)
			}
			if !tt.wantBlock && result.Action != guardrails.ActionAllow {
				t.Errorf("[%s] expected allow, got %v", tt.reason, result.Action)
			}
		})
	}
}

func TestArgumentValidator_BlockedArgs(t *testing.T) {
	ctx := context.Background()
	validator := tool.NewArgumentValidator().
		BlockArgs("execute", "privileged", "force")

	tests := []struct {
		name      string
		toolName  string
		args      map[string]any
		wantBlock bool
	}{
		{
			name:      "no blocked args",
			toolName:  "execute",
			args:      map[string]any{"command": "ls"},
			wantBlock: false,
		},
		{
			name:      "has privileged arg",
			toolName:  "execute",
			args:      map[string]any{"command": "ls", "privileged": true},
			wantBlock: true,
		},
		{
			name:      "has force arg",
			toolName:  "execute",
			args:      map[string]any{"command": "rm", "force": true},
			wantBlock: true,
		},
		{
			name:      "different tool same arg name allowed",
			toolName:  "other_tool",
			args:      map[string]any{"privileged": true},
			wantBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Check(ctx, tt.toolName, tt.args)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantBlock && result.Action != guardrails.ActionBlock {
				t.Errorf("expected block, got %v", result.Action)
			}
			if !tt.wantBlock && result.Action != guardrails.ActionAllow {
				t.Errorf("expected allow, got %v", result.Action)
			}
		})
	}
}

func TestArgumentValidator_CombinedRequiredAndBlocked(t *testing.T) {
	ctx := context.Background()
	validator := tool.NewArgumentValidator().
		RequireArgs("execute", "command").
		BlockArgs("execute", "sudo")

	tests := []struct {
		name      string
		args      map[string]any
		wantBlock bool
		reason    string
	}{
		{
			name:      "valid call",
			args:      map[string]any{"command": "ls"},
			wantBlock: false,
		},
		{
			name:      "missing required",
			args:      map[string]any{},
			wantBlock: true,
			reason:    "missing command",
		},
		{
			name:      "has blocked arg",
			args:      map[string]any{"command": "ls", "sudo": true},
			wantBlock: true,
			reason:    "sudo is blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := validator.Check(ctx, "execute", tt.args)
			if tt.wantBlock && result.Action != guardrails.ActionBlock {
				t.Errorf("[%s] expected block, got %v", tt.reason, result.Action)
			}
			if !tt.wantBlock && result.Action != guardrails.ActionAllow {
				t.Errorf("[%s] expected allow, got %v", tt.reason, result.Action)
			}
		})
	}
}

func TestArgumentValidator_Details(t *testing.T) {
	ctx := context.Background()
	validator := tool.NewArgumentValidator().
		RequireArgs("search", "query")

	result, _ := validator.Check(ctx, "search", map[string]any{})

	if result.Details == nil {
		t.Fatal("Details should not be nil")
	}
	if result.Details["tool"] != "search" {
		t.Errorf("Details[tool] = %v, want search", result.Details["tool"])
	}
	if result.Details["argument"] != "query" {
		t.Errorf("Details[argument] = %v, want query", result.Details["argument"])
	}
}

func TestArgumentValidator_NilArgs(t *testing.T) {
	ctx := context.Background()
	validator := tool.NewArgumentValidator().
		RequireArgs("search", "query")

	// nil args should be treated as empty map
	result, err := validator.Check(ctx, "search", nil)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Action != guardrails.ActionBlock {
		t.Error("nil args with required fields should block")
	}
}

func TestArgumentValidator_InterfaceCompliance(t *testing.T) {
	var _ guardrails.ToolGuardrail = tool.NewArgumentValidator()
}
