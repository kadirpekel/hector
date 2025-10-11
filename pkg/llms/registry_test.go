package llms

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a"
)

func TestNewLLMRegistry(t *testing.T) {
	registry := NewLLMRegistry()
	if registry == nil {
		t.Fatal("NewLLMRegistry() returned nil")
	}

	// Test that registry is initialized
	providers := registry.BaseRegistry.List()
	if providers == nil {
		t.Error("List() should not return nil")
	}
}

func TestLLMRegistry_RegisterLLM(t *testing.T) {
	registry := NewLLMRegistry()

	// Create a mock provider
	provider := &MockLLMProvider{
		name:  "test-provider",
		model: "test-model",
	}

	err := registry.RegisterLLM("test-provider", provider)
	if err != nil {
		t.Fatalf("RegisterLLM() error = %v", err)
	}

	// Verify provider was registered
	registeredProvider, exists := registry.BaseRegistry.Get("test-provider")
	if !exists {
		t.Error("Expected provider to be registered")
	}
	if registeredProvider != provider {
		t.Error("Expected registered provider to match")
	}
}

func TestLLMRegistry_RegisterLLM_Duplicate(t *testing.T) {
	registry := NewLLMRegistry()

	provider := &MockLLMProvider{name: "test-provider"}

	// Register first time
	err := registry.RegisterLLM("test-provider", provider)
	if err != nil {
		t.Fatalf("RegisterLLM() error = %v", err)
	}

	// Try to register again
	err = registry.RegisterLLM("test-provider", provider)
	if err == nil {
		t.Error("Expected error when registering duplicate provider")
	}
}

func TestLLMRegistry_Get(t *testing.T) {
	registry := NewLLMRegistry()

	provider := &MockLLMProvider{
		name:  "test-provider",
		model: "test-model",
	}

	err := registry.RegisterLLM("test-provider", provider)
	if err != nil {
		t.Fatalf("RegisterLLM() error = %v", err)
	}

	// Get the provider
	registeredProvider, exists := registry.BaseRegistry.Get("test-provider")
	if !exists {
		t.Fatal("Get() should return true for existing provider")
	}

	if registeredProvider == nil {
		t.Fatal("Get() returned nil")
	}

	if registeredProvider.GetModelName() != "test-model" {
		t.Errorf("Get() model = %v, want 'test-model'", registeredProvider.GetModelName())
	}
}

func TestLLMRegistry_Get_NotFound(t *testing.T) {
	registry := NewLLMRegistry()

	_, exists := registry.BaseRegistry.Get("non-existent-provider")
	if exists {
		t.Error("Expected false when getting non-existent provider")
	}
}

func TestLLMRegistry_List(t *testing.T) {
	registry := NewLLMRegistry()

	// Initially should be empty
	providers := registry.BaseRegistry.List()
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers initially, got %d", len(providers))
	}

	// Register a provider
	provider := &MockLLMProvider{name: "test-provider"}
	err := registry.RegisterLLM("test-provider", provider)
	if err != nil {
		t.Fatalf("RegisterLLM() error = %v", err)
	}

	// Should now have one provider
	providers = registry.BaseRegistry.List()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}
}

func TestLLMRegistry_Remove(t *testing.T) {
	registry := NewLLMRegistry()

	// Register a provider
	provider := &MockLLMProvider{name: "test-provider"}
	err := registry.RegisterLLM("test-provider", provider)
	if err != nil {
		t.Fatalf("RegisterLLM() error = %v", err)
	}

	// Remove the provider
	err = registry.BaseRegistry.Remove("test-provider")
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// Verify provider was removed
	_, exists := registry.BaseRegistry.Get("test-provider")
	if exists {
		t.Error("Expected provider to be removed")
	}
}

func TestLLMRegistry_Remove_NotFound(t *testing.T) {
	registry := NewLLMRegistry()

	err := registry.BaseRegistry.Remove("non-existent-provider")
	if err == nil {
		t.Error("Expected error when removing non-existent provider")
	}
}

func TestLLMRegistry_Count(t *testing.T) {
	registry := NewLLMRegistry()

	// Initially should be 0
	count := registry.BaseRegistry.Count()
	if count != 0 {
		t.Errorf("Expected count 0 initially, got %d", count)
	}

	// Register providers
	provider1 := &MockLLMProvider{name: "provider1"}
	provider2 := &MockLLMProvider{name: "provider2"}

	_ = registry.RegisterLLM("provider1", provider1)
	_ = registry.RegisterLLM("provider2", provider2)

	// Should now be 2
	count = registry.BaseRegistry.Count()
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestLLMRegistry_Clear(t *testing.T) {
	registry := NewLLMRegistry()

	// Register a provider
	provider := &MockLLMProvider{name: "test-provider"}
	_ = registry.RegisterLLM("test-provider", provider)

	// Clear the registry
	registry.BaseRegistry.Clear()

	// Should be empty
	count := registry.BaseRegistry.Count()
	if count != 0 {
		t.Errorf("Expected count 0 after clear, got %d", count)
	}
}

// Mock implementations for testing
type MockLLMProvider struct {
	name  string
	model string
}

func (m *MockLLMProvider) Generate(messages []a2a.Message, tools []ToolDefinition) (string, []a2a.ToolCall, int, error) {
	return "Mock response", []a2a.ToolCall{}, 10, nil
}

func (m *MockLLMProvider) GenerateStreaming(messages []a2a.Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	ch <- StreamChunk{Text: "Mock streaming response"}
	close(ch)
	return ch, nil
}

func (m *MockLLMProvider) GetModelName() string {
	return m.model
}

func (m *MockLLMProvider) GetMaxTokens() int {
	return 1000
}

func (m *MockLLMProvider) GetTemperature() float64 {
	return 0.7
}

func (m *MockLLMProvider) Close() error {
	return nil
}
