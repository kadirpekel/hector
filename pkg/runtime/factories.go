package runtime

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/builder"
	"github.com/verikod/hector/pkg/config"
	"github.com/verikod/hector/pkg/embedder"
	"github.com/verikod/hector/pkg/memory"
	"github.com/verikod/hector/pkg/model"
	"github.com/verikod/hector/pkg/tool/functiontool"
	"github.com/verikod/hector/pkg/tool"
	"github.com/verikod/hector/pkg/tool/commandtool"
	"github.com/verikod/hector/pkg/tool/filetool"
	"github.com/verikod/hector/pkg/tool/todotool"
	"github.com/verikod/hector/pkg/tool/webtool"
	"gopkg.in/yaml.v3"
)

// DefaultLLMFactory creates LLM instances based on provider type.
// Uses the builder package as its foundation to ensure a single code path
// for both configuration-based and programmatic API usage.
func DefaultLLMFactory(cfg *config.LLMConfig) (model.LLM, error) {
	return builder.LLMFromConfig(cfg).Build()
}

// DefaultEmbedderFactory creates Embedder instances based on provider type.
// Uses the builder package as its foundation to ensure a single code path
// for both configuration-based and programmatic API usage.
func DefaultEmbedderFactory(cfg *config.EmbedderConfig) (embedder.Embedder, error) {
	return builder.EmbedderFromConfig(cfg).Build()
}

// DefaultToolsetFactory creates toolset instances based on tool type.
// Uses the builder package as its foundation for MCP toolsets to ensure
// a single code path for both configuration-based and programmatic API usage.
func DefaultToolsetFactory(name string, cfg *config.ToolConfig) (tool.Toolset, error) {
	switch cfg.Type {
	case config.ToolTypeMCP:
		// Use builder as foundation for MCP toolsets
		return builder.MCPFromConfig(name, cfg).Build()

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
	case "text_editor":
		// Use defaults - options can be provided via config
		t, err = filetool.NewTextEditor(nil)

	case "apply_patch":
		// Use defaults
		t, err = filetool.NewApplyPatch(nil)

	case "grep_search":
		// Use defaults
		t, err = filetool.NewGrepSearch(nil)

	case "web_request":
		// Use defaults
		t, err = webtool.NewWebRequest(nil)

	case "web_fetch":
		// Use defaults
		t, err = webtool.NewWebFetch(nil)

	case "web_search":
		// Use defaults
		t, err = webtool.NewWebSearch(nil)

	case "todo_write":
		// TodoManager is stateless - create a new one for each toolset
		todoManager := todotool.NewTodoManager()
		t, err = todoManager.Tool()

	case "config_validator":
		t, err = newConfigValidatorTool()

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

type validateConfigArgs struct {
	YAML string `json:"yaml" jsonschema:"required,description=Hector AppConfig YAML text to validate"`
}

func newConfigValidatorTool() (tool.CallableTool, error) {
	return functiontool.New(functiontool.Config{
		Name:        "config_validator",
		Description: "Validate Hector AppConfig YAML and return normalized JSON plus errors",
	}, func(ctx tool.Context, args validateConfigArgs) (map[string]any, error) {
		var cfg config.AppConfig
		if err := yaml.Unmarshal([]byte(args.YAML), &cfg); err != nil {
			return map[string]any{
				"ok":    false,
				"error": fmt.Sprintf("invalid yaml: %v", err),
			}, nil
		}

		cfg.SetDefaults()
		if err := config.ValidateAppConfig(&cfg); err != nil {
			return map[string]any{
				"ok":    false,
				"error": err.Error(),
			}, nil
		}

		normalized, err := json.MarshalIndent(&cfg, "", "  ")
		if err != nil {
			return map[string]any{
				"ok":    true,
				"error": fmt.Sprintf("validation succeeded but normalization failed: %v", err),
			}, nil
		}

		return map[string]any{
			"ok":              true,
			"normalized_json": string(normalized),
		}, nil
	})
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
// Uses the builder package as its foundation to ensure a single code path
// for both configuration-based and programmatic API usage.
// Returns nil if no context config is set or strategy is "none".
func DefaultWorkingMemoryFactory(opts WorkingMemoryFactoryOptions) (memory.WorkingMemoryStrategy, error) {
	cfg := opts.Config
	if cfg == nil {
		return nil, nil // No context config, no filtering
	}

	cfg.SetDefaults()

	if cfg.Strategy == "" || cfg.Strategy == "none" {
		return nil, nil // No filtering
	}

	// Use builder as foundation
	b := builder.WorkingMemoryFromConfig(cfg).ModelName(opts.ModelName)

	// Set summarizer LLM for summary_buffer strategy
	if cfg.Strategy == "summary_buffer" && opts.SummarizerLLM != nil {
		b = b.WithLLM(opts.SummarizerLLM)
	}

	return b.Build()
}
