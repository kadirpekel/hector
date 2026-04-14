package builder_test

import (
	"testing"

	"github.com/verikod/hector/pkg/builder"
	"github.com/verikod/hector/pkg/tool"
)

// =============================================================================
// CredentialsBuilder Tests
// =============================================================================

func TestNewCredentials(t *testing.T) {
	b := builder.NewCredentials()
	if b == nil {
		t.Fatal("NewCredentials() returned nil")
	}

	// Default type should be bearer
	creds := b.Build()
	if creds.Type != "bearer" {
		t.Errorf("Default Type = %q, want bearer", creds.Type)
	}
	if creds.APIKeyHeader != "X-API-Key" {
		t.Errorf("Default APIKeyHeader = %q, want X-API-Key", creds.APIKeyHeader)
	}
}

func TestCredentialsBuilder_BearerToken(t *testing.T) {
	creds := builder.NewCredentials().
		Type("bearer").
		Token("my-secret-token").
		Build()

	if creds.Type != "bearer" {
		t.Errorf("Type = %q, want bearer", creds.Type)
	}
	if creds.Token != "my-secret-token" {
		t.Errorf("Token = %q, want my-secret-token", creds.Token)
	}
}

func TestCredentialsBuilder_APIKey(t *testing.T) {
	creds := builder.NewCredentials().
		Type("api_key").
		APIKey("my-api-key").
		APIKeyHeader("Authorization").
		Build()

	if creds.Type != "api_key" {
		t.Errorf("Type = %q, want api_key", creds.Type)
	}
	if creds.APIKey != "my-api-key" {
		t.Errorf("APIKey = %q, want my-api-key", creds.APIKey)
	}
	if creds.APIKeyHeader != "Authorization" {
		t.Errorf("APIKeyHeader = %q, want Authorization", creds.APIKeyHeader)
	}
}

func TestCredentialsBuilder_Basic(t *testing.T) {
	creds := builder.NewCredentials().
		Type("basic").
		Username("admin").
		Password("secret123").
		Build()

	if creds.Type != "basic" {
		t.Errorf("Type = %q, want basic", creds.Type)
	}
	if creds.Username != "admin" {
		t.Errorf("Username = %q, want admin", creds.Username)
	}
	if creds.Password != "secret123" {
		t.Errorf("Password = %q, want secret123", creds.Password)
	}
}

func TestCredentialsBuilder_FluentChaining(t *testing.T) {
	// Verify each method returns the builder for chaining
	b := builder.NewCredentials()
	result := b.Type("bearer").Token("token").APIKey("key").Username("user").Password("pass")

	if result == nil {
		t.Error("Fluent chaining should return builder")
	}
}

// =============================================================================
// ReasoningBuilder Tests
// =============================================================================

func TestNewReasoning(t *testing.T) {
	b := builder.NewReasoning()
	if b == nil {
		t.Fatal("NewReasoning() returned nil")
	}

	// Default max iterations should be 100
	cfg := b.Build()
	if cfg.MaxIterations != 100 {
		t.Errorf("Default MaxIterations = %d, want 100", cfg.MaxIterations)
	}
}

func TestReasoningBuilder_MaxIterations(t *testing.T) {
	cfg := builder.NewReasoning().
		MaxIterations(50).
		Build()

	if cfg.MaxIterations != 50 {
		t.Errorf("MaxIterations = %d, want 50", cfg.MaxIterations)
	}
}

func TestReasoningBuilder_MaxIterations_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MaxIterations(0) should panic")
		}
	}()

	builder.NewReasoning().MaxIterations(0)
}

func TestReasoningBuilder_MaxIterations_NegativePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MaxIterations(-1) should panic")
		}
	}()

	builder.NewReasoning().MaxIterations(-1)
}

func TestReasoningBuilder_EnableExitTool(t *testing.T) {
	cfg := builder.NewReasoning().
		EnableExitTool(true).
		Build()

	if !cfg.EnableExitTool {
		t.Error("EnableExitTool should be true")
	}
}

func TestReasoningBuilder_EnableEscalateTool(t *testing.T) {
	cfg := builder.NewReasoning().
		EnableEscalateTool(true).
		Build()

	if !cfg.EnableEscalateTool {
		t.Error("EnableEscalateTool should be true")
	}
}

func TestReasoningBuilder_CompletionInstruction(t *testing.T) {
	instruction := "Call exit_loop when you have a final answer."
	cfg := builder.NewReasoning().
		CompletionInstruction(instruction).
		Build()

	if cfg.CompletionInstruction != instruction {
		t.Errorf("CompletionInstruction = %q, want %q", cfg.CompletionInstruction, instruction)
	}
}

func TestReasoningBuilder_FullConfiguration(t *testing.T) {
	cfg := builder.NewReasoning().
		MaxIterations(25).
		EnableExitTool(true).
		EnableEscalateTool(true).
		CompletionInstruction("Done when finished.").
		Build()

	if cfg.MaxIterations != 25 {
		t.Error("MaxIterations should be 25")
	}
	if !cfg.EnableExitTool {
		t.Error("EnableExitTool should be true")
	}
	if !cfg.EnableEscalateTool {
		t.Error("EnableEscalateTool should be true")
	}
	if cfg.CompletionInstruction != "Done when finished." {
		t.Error("CompletionInstruction mismatch")
	}
}

func TestReasoningBuilder_FluentChaining(t *testing.T) {
	b := builder.NewReasoning()
	result := b.MaxIterations(50).EnableExitTool(true).EnableEscalateTool(false)

	if result == nil {
		t.Error("Fluent chaining should return builder")
	}
}

// =============================================================================
// FunctionTool Tests
// =============================================================================

type testArgs struct {
	Name string `json:"name" jsonschema:"required,description=The name"`
}

func TestFunctionTool(t *testing.T) {
	fn := func(ctx tool.Context, args testArgs) (map[string]any, error) {
		return map[string]any{"result": args.Name}, nil
	}

	tool, err := builder.FunctionTool("test_tool", "A test tool", fn)

	if err != nil {
		t.Fatalf("FunctionTool() error = %v", err)
	}
	if tool == nil {
		t.Fatal("FunctionTool() returned nil")
	}
	if tool.Name() != "test_tool" {
		t.Errorf("Name() = %q, want test_tool", tool.Name())
	}
	if tool.Description() != "A test tool" {
		t.Errorf("Description() = %q, want 'A test tool'", tool.Description())
	}
}

func TestMustFunctionTool(t *testing.T) {
	fn := func(ctx tool.Context, args testArgs) (map[string]any, error) {
		return nil, nil
	}

	// Should not panic for valid configuration
	tool := builder.MustFunctionTool("must_tool", "A must tool", fn)

	if tool == nil {
		t.Fatal("MustFunctionTool() returned nil")
	}
	if tool.Name() != "must_tool" {
		t.Errorf("Name() = %q", tool.Name())
	}
}
