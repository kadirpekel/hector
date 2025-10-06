package reasoning

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
)

func TestCreateStrategy(t *testing.T) {
	tests := []struct {
		name            string
		engine          string
		reasoningConfig *config.ReasoningConfig
		wantErr         bool
	}{
		{
			name:   "chain_of_thought strategy",
			engine: "chain-of-thought",
			reasoningConfig: &config.ReasoningConfig{
				Engine: "chain-of-thought",
			},
			wantErr: false,
		},
		{
			name:   "supervisor strategy",
			engine: "supervisor",
			reasoningConfig: &config.ReasoningConfig{
				Engine: "supervisor",
			},
			wantErr: false,
		},
		{
			name:   "unknown strategy",
			engine: "unknown",
			reasoningConfig: &config.ReasoningConfig{
				Engine: "unknown",
			},
			wantErr: true,
		},
		{
			name:            "empty engine",
			engine:          "",
			reasoningConfig: &config.ReasoningConfig{},
			wantErr:         false, // Empty engine defaults to chain-of-thought
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy, err := CreateStrategy(tt.engine, *tt.reasoningConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateStrategy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && strategy == nil {
				t.Error("CreateStrategy() returned nil strategy without error")
			}
		})
	}
}

func TestListAvailableStrategies(t *testing.T) {
	strategies := ListAvailableStrategies()

	if len(strategies) == 0 {
		t.Error("ListAvailableStrategies() should return at least one strategy")
	}

	// Check that we have the expected strategies
	expectedStrategies := []string{"chain-of-thought", "supervisor"}
	for _, expected := range expectedStrategies {
		found := false
		for _, strategy := range strategies {
			if strategy.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ListAvailableStrategies() missing expected strategy: %s", expected)
		}
	}
}
