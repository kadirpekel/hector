package moderation

import (
	"fmt"

	"github.com/verikod/hector/pkg/model"
)

// Config configures the moderation provider to use.
type Config struct {
	// Strategy determines which moderation approach to use.
	// Values: "none", "openai", "lakera", "prompt", "agent"
	Strategy Strategy

	// OpenAI configures the OpenAI moderation provider.
	OpenAI *OpenAIConfig

	// Lakera configures the Lakera Guard provider.
	Lakera *LakeraConfig

	// Prompt configures the prompt-based provider.
	Prompt *PromptConfig
}

// Build creates a Provider from the configuration.
// The llm parameter is used for prompt-based moderation if needed.
func Build(cfg *Config, llm model.LLM) (Provider, error) {
	if cfg == nil || cfg.Strategy == "" || cfg.Strategy == StrategyNone {
		return nil, nil
	}

	switch cfg.Strategy {
	case StrategyOpenAI:
		openaiCfg := cfg.OpenAI
		if openaiCfg == nil {
			openaiCfg = &OpenAIConfig{}
		}
		return NewOpenAI(*openaiCfg), nil

	case StrategyLakera:
		lakeraCfg := cfg.Lakera
		if lakeraCfg == nil {
			lakeraCfg = &LakeraConfig{}
		}
		return NewLakera(*lakeraCfg), nil

	case StrategyPrompt:
		promptCfg := cfg.Prompt
		if promptCfg == nil {
			promptCfg = &PromptConfig{}
		}
		if promptCfg.LLM == nil {
			promptCfg.LLM = llm
		}
		if promptCfg.LLM == nil {
			return nil, fmt.Errorf("prompt moderation requires an LLM")
		}
		return NewPrompt(*promptCfg), nil

	case StrategyAgent:
		// Agent strategy should use conditional workflow agent instead
		return nil, fmt.Errorf("agent strategy: use conditional workflow agent (type: conditional) instead")

	default:
		return nil, fmt.Errorf("unknown moderation strategy: %s", cfg.Strategy)
	}
}
