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

package runtime

import (
	"fmt"
	"time"

	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/embedder"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/kadirpekel/hector/pkg/model"
	"github.com/kadirpekel/hector/pkg/model/anthropic"
	"github.com/kadirpekel/hector/pkg/model/gemini"
	"github.com/kadirpekel/hector/pkg/model/ollama"
	"github.com/kadirpekel/hector/pkg/model/openai"
	"github.com/kadirpekel/hector/pkg/tool"
	"github.com/kadirpekel/hector/pkg/tool/commandtool"
	"github.com/kadirpekel/hector/pkg/tool/filetool"
	"github.com/kadirpekel/hector/pkg/tool/mcptoolset"
	"github.com/kadirpekel/hector/pkg/tool/todotool"
	"github.com/kadirpekel/hector/pkg/tool/webtool"
)

// DefaultLLMFactory creates LLM instances based on provider type.
func DefaultLLMFactory(cfg *config.LLMConfig) (model.LLM, error) {
	switch cfg.Provider {
	case config.LLMProviderAnthropic:
		acfg := anthropic.Config{
			APIKey:      cfg.APIKey,
			Model:       cfg.Model,
			MaxTokens:   cfg.MaxTokens,
			Temperature: cfg.Temperature,
			BaseURL:     cfg.BaseURL,
		}
		if cfg.Thinking != nil && config.BoolValue(cfg.Thinking.Enabled, false) {
			acfg.EnableThinking = true
			acfg.ThinkingBudget = cfg.Thinking.BudgetTokens
		}
		return anthropic.New(acfg)

	case config.LLMProviderOpenAI:
		ocfg := openai.Config{
			APIKey:      cfg.APIKey,
			Model:       cfg.Model,
			MaxTokens:   cfg.MaxTokens,
			Temperature: cfg.Temperature,
			BaseURL:     cfg.BaseURL,
		}
		if cfg.Thinking != nil && config.BoolValue(cfg.Thinking.Enabled, false) {
			ocfg.EnableReasoning = true
			ocfg.ReasoningBudget = cfg.Thinking.BudgetTokens
		}
		return openai.New(ocfg)

	case config.LLMProviderGemini:
		gcfg := gemini.Config{
			APIKey:    cfg.APIKey,
			Model:     cfg.Model,
			MaxTokens: cfg.MaxTokens,
		}
		if cfg.Temperature != nil {
			gcfg.Temperature = *cfg.Temperature
		}
		return gemini.New(gcfg)

	case config.LLMProviderOllama:
		ocfg := ollama.Config{
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}
		if cfg.Temperature != nil {
			ocfg.Temperature = cfg.Temperature
		}
		if cfg.MaxTokens > 0 {
			numPredict := cfg.MaxTokens
			ocfg.NumPredict = &numPredict
		}
		if cfg.Thinking != nil && config.BoolValue(cfg.Thinking.Enabled, false) {
			ocfg.EnableThinking = true
		}
		return ollama.New(ocfg)

	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}

// DefaultEmbedderFactory creates Embedder instances based on provider type.
func DefaultEmbedderFactory(cfg *config.EmbedderConfig) (embedder.Embedder, error) {
	return embedder.NewEmbedderFromConfig(cfg)
}

// DefaultToolsetFactory creates toolset instances based on tool type.
func DefaultToolsetFactory(name string, cfg *config.ToolConfig) (tool.Toolset, error) {
	switch cfg.Type {
	case config.ToolTypeMCP:
		return mcptoolset.New(mcptoolset.Config{
			Name:      name,
			URL:       cfg.URL,
			Transport: cfg.Transport,
			Command:   cfg.Command,
			Args:      cfg.Args,
			Env:       cfg.Env,
			Filter:    cfg.Filter,
		})

	case config.ToolTypeCommand:
		// Build command tool configuration
		cmdCfg := commandtool.Config{
			Name:            name,
			AllowedCommands: cfg.AllowedCommands,
			DeniedCommands:  cfg.DeniedCommands,
			WorkingDir:      cfg.WorkingDirectory,
		}

		// Parse timeout
		if cfg.MaxExecutionTime != "" {
			duration, err := time.ParseDuration(cfg.MaxExecutionTime)
			if err != nil {
				return nil, fmt.Errorf("invalid max_execution_time: %w", err)
			}
			cmdCfg.Timeout = duration
		}

		// HITL settings
		if cfg.RequireApproval != nil && *cfg.RequireApproval {
			cmdCfg.RequireApproval = true
		}
		if cfg.ApprovalPrompt != "" {
			cmdCfg.ApprovalPrompt = cfg.ApprovalPrompt
		}
		if cfg.DenyByDefault != nil && *cfg.DenyByDefault {
			cmdCfg.DenyByDefault = true
		}

		// Wrap standalone tool in a toolset
		cmdTool := commandtool.New(cmdCfg)
		return &singleToolset{name: name, tool: cmdTool}, nil

	case config.ToolTypeFunction:
		return createFunctionToolset(name, cfg)

	default:
		return nil, fmt.Errorf("unknown tool type: %s", cfg.Type)
	}
}

// singleToolset wraps a standalone tool as a toolset.
type singleToolset struct {
	name string
	tool tool.Tool
}

func (s *singleToolset) Name() string {
	return s.name
}

func (s *singleToolset) Tools(ctx agent.ReadonlyContext) ([]tool.Tool, error) {
	return []tool.Tool{s.tool}, nil
}

// createFunctionToolset creates a function toolset based on handler name.
// Uses default configs for tool-specific settings since ToolConfig only has common fields.
// Tool-specific configuration can be added to ToolConfig in the future if needed.
func createFunctionToolset(name string, cfg *config.ToolConfig) (tool.Toolset, error) {
	if cfg.Handler == "" {
		return nil, fmt.Errorf("function tool requires handler")
	}

	// Create tool based on handler name
	// Most tools use nil config to get defaults, since ToolConfig doesn't have
	// tool-specific fields (those would come from config file if needed)
	var t tool.CallableTool
	var err error

	switch cfg.Handler {
	case "read_file":
		// Use defaults - MaxFileSize and WorkingDirectory can be set via config file if needed
		t, err = filetool.NewReadFile(nil)

	case "write_file":
		// Use defaults - file writing is risky but tool handles it
		t, err = filetool.NewWriteFile(nil)

	case "search_replace":
		// Use defaults
		t, err = filetool.NewSearchReplace(nil)

	case "apply_patch":
		// Use defaults
		t, err = filetool.NewApplyPatch(nil)

	case "grep_search":
		// Use defaults
		t, err = filetool.NewGrepSearch(nil)

	case "web_request":
		// Use defaults
		t, err = webtool.NewWebRequest(nil)

	case "todo_write":
		// TodoManager is stateless - create a new one for each toolset
		todoManager := todotool.NewTodoManager()
		t, err = todoManager.Tool()

	default:
		return nil, fmt.Errorf("unknown function tool handler: %s", cfg.Handler)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create function tool %q: %w", cfg.Handler, err)
	}

	// Wrap with approval required (HITL) if configured
	// This makes RequiresApproval() return true, triggering the HITL flow in agent
	if config.BoolValue(cfg.RequireApproval, false) {
		t = withApprovalRequired(t, cfg.ApprovalPrompt)
	}

	// Wrap in toolset
	return &singleToolset{name: name, tool: t}, nil
}

// approvalRequiredTool wraps a CallableTool to return RequiresApproval() = true.
// This is used for tools that need HITL (human-in-the-loop) approval before execution.
// The actual HITL flow is handled by the agent flow, not the tool.
type approvalRequiredTool struct {
	tool.CallableTool
	approvalPrompt string
}

func (t *approvalRequiredTool) RequiresApproval() bool {
	return true
}

// ApprovalPrompt returns the custom approval prompt if set.
func (t *approvalRequiredTool) ApprovalPrompt() string {
	return t.approvalPrompt
}

func withApprovalRequired(t tool.CallableTool, approvalPrompt string) tool.CallableTool {
	return &approvalRequiredTool{
		CallableTool:   t,
		approvalPrompt: approvalPrompt,
	}
}

// WorkingMemoryFactoryOptions contains options for creating working memory strategies.
type WorkingMemoryFactoryOptions struct {
	// Config is the context configuration.
	Config *config.ContextConfig

	// ModelName is the LLM model name for token counting.
	ModelName string

	// SummarizerLLM is the LLM to use for summarization (summary_buffer only).
	// If nil and strategy is summary_buffer, summarization is disabled.
	SummarizerLLM model.LLM
}

// DefaultWorkingMemoryFactory creates a working memory strategy from config.
// Returns nil if no context config is set or strategy is "none".
func DefaultWorkingMemoryFactory(opts WorkingMemoryFactoryOptions) (memory.WorkingMemoryStrategy, error) {
	cfg := opts.Config
	if cfg == nil {
		return nil, nil // No context config, no filtering
	}

	cfg.SetDefaults()

	switch cfg.Strategy {
	case "", "none":
		return nil, nil // No filtering

	case "buffer_window":
		return memory.NewBufferWindowStrategy(memory.BufferWindowConfig{
			WindowSize: cfg.WindowSize,
		}), nil

	case "token_window":
		strategy, err := memory.NewTokenWindowStrategy(memory.TokenWindowConfig{
			Budget:         cfg.Budget,
			PreserveRecent: cfg.PreserveRecent,
			Model:          opts.ModelName,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create token_window strategy: %w", err)
		}
		return strategy, nil

	case "summary_buffer":
		// Create summarizer if LLM is provided
		var summarizer memory.Summarizer
		if opts.SummarizerLLM != nil {
			var err error
			summarizer, err = memory.NewLLMSummarizer(memory.LLMSummarizerConfig{
				LLM: opts.SummarizerLLM,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create summarizer: %w", err)
			}
		}

		strategy, err := memory.NewSummaryBufferStrategy(memory.SummaryBufferConfig{
			Budget:     cfg.Budget,
			Threshold:  cfg.Threshold,
			Target:     cfg.Target,
			Model:      opts.ModelName,
			Summarizer: summarizer,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create summary_buffer strategy: %w", err)
		}
		return strategy, nil

	default:
		return nil, fmt.Errorf("unknown context strategy: %s", cfg.Strategy)
	}
}
