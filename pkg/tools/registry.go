package tools

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/observability"
	"github.com/kadirpekel/hector/pkg/registry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ToolEntry struct {
	Tool       Tool       `json:"tool"`
	Source     ToolSource `json:"source"`
	SourceType string     `json:"source_type"`
	Name       string     `json:"name"`
}

type ToolRegistryError struct {
	Component string
	Action    string
	Message   string
	Err       error
}

func (e *ToolRegistryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Component, e.Action, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Component, e.Action, e.Message)
}

func NewToolRegistryError(component, action, message string, err error) *ToolRegistryError {
	return &ToolRegistryError{
		Component: component,
		Action:    action,
		Message:   message,
		Err:       err,
	}
}

type ToolRegistry struct {
	*registry.BaseRegistry[ToolEntry]
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		BaseRegistry: registry.NewBaseRegistry[ToolEntry](),
	}
}

func NewToolRegistryWithConfig(toolConfig map[string]*config.ToolConfig) (*ToolRegistry, error) {
	return NewToolRegistryWithConfigAndAgentRegistry(toolConfig, nil)
}

func NewToolRegistryWithConfigAndAgentRegistry(toolConfig map[string]*config.ToolConfig, agentRegistry interface{}) (*ToolRegistry, error) {
	registry := &ToolRegistry{
		BaseRegistry: registry.NewBaseRegistry[ToolEntry](),
	}

	if toolConfig != nil {
		if err := registry.initializeFromConfigWithAgentRegistry(toolConfig, agentRegistry); err != nil {
			return nil, fmt.Errorf("failed to initialize tool registry from config: %w", err)
		}
	}

	return registry, nil
}

func (r *ToolRegistry) RegisterSource(source ToolSource) error {
	name := source.GetName()
	if name == "" {
		return NewToolRegistryError("ToolRegistry", "RegisterSource", "source name cannot be empty", nil)
	}

	if err := source.DiscoverTools(context.Background()); err != nil {
		return NewToolRegistryError("ToolRegistry", "RegisterSource",
			fmt.Sprintf("failed to discover tools from source %s", name), err)
	}

	for _, toolInfo := range source.ListTools() {
		tool, exists := source.GetTool(toolInfo.Name)
		if !exists {
			continue
		}

		entry := ToolEntry{
			Tool:       tool,
			Source:     source,
			SourceType: source.GetType(),
			Name:       toolInfo.Name,
		}

		if err := r.Register(toolInfo.Name, entry); err != nil {
			return NewToolRegistryError("ToolRegistry", "RegisterSource",
				fmt.Sprintf("failed to register tool %s", toolInfo.Name), err)
		}
	}

	return nil
}

func (r *ToolRegistry) DiscoverAllTools(ctx context.Context) error {

	repositories := make(map[string]ToolSource)
	for _, entry := range r.List() {
		repositories[entry.Source.GetName()] = entry.Source
	}

	r.Clear()

	for repoName, repo := range repositories {
		if err := repo.DiscoverTools(ctx); err != nil {
			fmt.Printf("Warning: Failed to discover tools from %s: %v\n", repoName, err)
			continue
		}

		for _, toolInfo := range repo.ListTools() {
			tool, exists := repo.GetTool(toolInfo.Name)
			if !exists {
				fmt.Printf("Warning: Tool %s listed but not available in %s\n", toolInfo.Name, repoName)
				continue
			}

			if _, exists := r.Get(toolInfo.Name); exists {
				fmt.Printf("Warning: Tool name conflict: %s already exists (skipping)\n", toolInfo.Name)
				continue
			}

			entry := ToolEntry{
				Tool:       tool,
				Source:     repo,
				SourceType: repo.GetType(),
				Name:       toolInfo.Name,
			}

			if err := r.Register(toolInfo.Name, entry); err != nil {
				return NewToolRegistryError("ToolRegistry", "DiscoverAllTools",
					fmt.Sprintf("failed to register tool %s", toolInfo.Name), err)
			}
		}
	}
	return nil
}

func (r *ToolRegistry) initializeFromConfigWithAgentRegistry(toolConfig map[string]*config.ToolConfig, agentRegistry interface{}) error {

	localTools := make(map[string]*config.ToolConfig)
	mcpTools := make(map[string]*config.ToolConfig)

	for name, tool := range toolConfig {
		if tool != nil {
			if tool.Type == "mcp" {
				mcpTools[name] = tool
			} else {
				localTools[name] = tool
			}
		}
	}

	if len(localTools) > 0 {
		repo, err := NewLocalToolSourceWithConfigAndAgentRegistry(localTools, agentRegistry)
		if err != nil {
			return fmt.Errorf("failed to create local tool source: %w", err)
		}

		if err := r.RegisterSource(repo); err != nil {
			return fmt.Errorf("failed to register local source: %w", err)
		}
	}

	for toolName, toolConfig := range mcpTools {
		if toolConfig == nil || !toolConfig.Enabled {
			continue
		}

		serverURL := toolConfig.ServerURL
		if serverURL == "" {
			fmt.Printf("Warning: MCP tool '%s' missing server_url, skipping\n", toolName)
			continue
		}

		mcpSource := NewMCPToolSource(toolName, serverURL, toolConfig.Description)

		if err := r.RegisterSource(mcpSource); err != nil {
			fmt.Printf("Warning: Failed to register MCP source '%s': %v\n", toolName, err)
			continue
		}
	}

	return nil
}

func (r *ToolRegistry) GetTool(name string) (Tool, error) {
	entry, exists := r.Get(name)
	if !exists {
		return nil, NewToolRegistryError("ToolRegistry", "GetTool",
			fmt.Sprintf("tool %s not found", name), nil)
	}
	return entry.Tool, nil
}

func (r *ToolRegistry) ListTools() []ToolInfo {
	var tools []ToolInfo
	for _, entry := range r.List() {
		info := entry.Tool.GetInfo()

		info.ServerURL = entry.Source.GetName()
		tools = append(tools, info)
	}

	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	return tools
}

func (r *ToolRegistry) ListToolsBySource() map[string][]ToolInfo {
	result := make(map[string][]ToolInfo)

	for _, entry := range r.List() {
		repoName := entry.Source.GetName()
		if result[repoName] == nil {
			result[repoName] = make([]ToolInfo, 0)
		}
		info := entry.Tool.GetInfo()
		result[repoName] = append(result[repoName], info)
	}

	return result
}

func (r *ToolRegistry) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (ToolResult, error) {
	startTime := time.Now()

	// Create span for tool execution
	tracer := observability.GetTracer("hector.tools")
	ctx, span := tracer.Start(ctx, observability.SpanToolExecution,
		trace.WithAttributes(
			attribute.String(observability.AttrToolName, toolName),
		),
	)
	defer span.End()

	tool, err := r.GetTool(toolName)
	if err != nil {
		// Record error in span
		span.RecordError(err)
		span.SetStatus(codes.Error, "tool not found")

		// Record metrics
		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordToolExecution(ctx, toolName, time.Since(startTime), err)
		}

		return ToolResult{
			Success:  false,
			Error:    err.Error(),
			ToolName: toolName,
		}, err
	}

	result, execErr := tool.Execute(ctx, args)
	duration := time.Since(startTime)

	// Record metrics and span status based on result
	metrics := observability.GetGlobalMetrics()
	if metrics != nil {
		var recordErr error
		if execErr != nil {
			// Execution error
			recordErr = execErr
			span.RecordError(execErr)
			span.SetStatus(codes.Error, execErr.Error())
		} else if !result.Success {
			// Tool returned failure
			recordErr = fmt.Errorf("%s", result.Error)
			span.RecordError(recordErr)
			span.SetStatus(codes.Error, result.Error)
		} else {
			span.SetStatus(codes.Ok, "success")
		}
		metrics.RecordToolExecution(ctx, toolName, duration, recordErr)
	}

	// Add result metadata to span
	span.SetAttributes(
		attribute.Bool("tool.success", result.Success),
		attribute.Int64("tool.duration_ms", duration.Milliseconds()),
	)

	return result, execErr
}

func (r *ToolRegistry) GetToolSource(toolName string) (string, error) {
	entry, exists := r.Get(toolName)
	if !exists {
		return "", NewToolRegistryError("ToolRegistry", "GetToolSource",
			fmt.Sprintf("tool %s not found", toolName), nil)
	}
	return entry.Source.GetName(), nil
}

func (r *ToolRegistry) RemoveSource(sourceName string) error {

	for _, entry := range r.List() {
		if entry.Source.GetName() == sourceName {
			if err := r.Remove(entry.Name); err != nil {
				return NewToolRegistryError("ToolRegistry", "RemoveSource",
					fmt.Sprintf("failed to remove tool %s", entry.Name), err)
			}
		}
	}

	return nil
}
