// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package llmagent provides an LLM-based agent implementation for Hector v2.
//
// LLM agents use language models to generate responses and can invoke tools
// to perform actions. They support:
//   - Instruction-based behavior control
//   - Tool/function calling
//   - Sub-agent delegation
//   - Callbacks for customization
//
// # Usage
//
//	agent, err := llmagent.New(llmagent.Config{
//	    Name:        "assistant",
//	    Model:       myModel,
//	    Instruction: "You are a helpful assistant.",
//	    Tools:       []tool.Tool{searchTool, calculatorTool},
//	})
package llmagent

import (
	"fmt"
	"iter"
	"log/slog"

	"github.com/a2aproject/a2a-go/a2a"

	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/kadirpekel/hector/pkg/model"
	"github.com/kadirpekel/hector/pkg/observability"
	"github.com/kadirpekel/hector/pkg/tool"
	"github.com/kadirpekel/hector/pkg/tool/controltool"
)

// Config contains the configuration for an LLM agent.
type Config struct {
	// Name must be unique within the agent tree.
	Name string

	// Description helps LLMs decide when to delegate to this agent.
	Description string

	// Model is the LLM to use for generation.
	Model model.LLM

	// Instruction guides the agent's behavior.
	// Supports template placeholders like {variable} resolved from state.
	Instruction string

	// EnableStreaming enables token-by-token streaming from the LLM.
	// When false (default), responses are returned as complete chunks.
	EnableStreaming bool

	// InstructionProvider allows dynamic instruction generation.
	// Takes precedence over Instruction if set.
	InstructionProvider InstructionProvider

	// GlobalInstruction applies to all agents in the tree.
	// Only the root agent's GlobalInstruction is used.
	GlobalInstruction string

	// GlobalInstructionProvider allows dynamic global instruction.
	GlobalInstructionProvider InstructionProvider

	// GenerateConfig contains LLM generation settings.
	GenerateConfig *model.GenerateConfig

	// Tools available to the agent.
	Tools []tool.Tool

	// Toolsets provide dynamic tool resolution.
	Toolsets []tool.Toolset

	// SubAgents can receive delegated tasks.
	SubAgents []agent.Agent

	// BeforeAgentCallbacks run before the agent starts.
	BeforeAgentCallbacks []agent.BeforeAgentCallback

	// AfterAgentCallbacks run after the agent completes.
	AfterAgentCallbacks []agent.AfterAgentCallback

	// BeforeModelCallbacks run before each LLM call.
	BeforeModelCallbacks []BeforeModelCallback

	// AfterModelCallbacks run after each LLM call.
	AfterModelCallbacks []AfterModelCallback

	// BeforeToolCallbacks run before each tool execution.
	BeforeToolCallbacks []BeforeToolCallback

	// AfterToolCallbacks run after each tool execution.
	AfterToolCallbacks []AfterToolCallback

	// DisallowTransferToParent prevents delegation to parent agent.
	DisallowTransferToParent bool

	// DisallowTransferToPeers prevents delegation to sibling agents.
	DisallowTransferToPeers bool

	// IncludeContents controls conversation history inclusion.
	IncludeContents IncludeContents

	// OutputKey saves agent output to session state under this key.
	OutputKey string

	// InputSchema validates input when agent is used as a tool.
	InputSchema map[string]any

	// OutputSchema enforces structured output format.
	OutputSchema map[string]any

	// Reasoning configures the chain-of-thought reasoning loop.
	// When nil, defaults are applied (semantic termination, 100 max iterations).
	Reasoning *ReasoningConfig

	// WorkingMemory is the context window management strategy.
	// Controls how conversation history is filtered to fit within LLM limits.
	// If nil, all history is included (no filtering).
	WorkingMemory memory.WorkingMemoryStrategy

	// ContextProvider retrieves relevant context for RAG.
	// When set, the agent will query the provider with user input
	// and inject relevant context into the conversation.
	ContextProvider ContextProvider

	// RequestProcessors are custom processors added to the request pipeline.
	// These run AFTER the default processors.
	RequestProcessors []RequestProcessor

	// ResponseProcessors are custom processors added to the response pipeline.
	// These run AFTER the default processors.
	ResponseProcessors []ResponseProcessor

	// Pipeline allows complete customization of the processor pipeline.
	// If set, RequestProcessors and ResponseProcessors are ignored.
	Pipeline *Pipeline

	// MetricsRecorder records tool execution metrics.
	// If nil, metrics are not recorded (no-op).
	MetricsRecorder observability.Recorder
}

// ReasoningConfig configures the chain-of-thought reasoning loop.
// This follows adk-go patterns for semantic loop termination.
type ReasoningConfig struct {
	// MaxIterations is a SAFETY limit (not the primary termination condition).
	// The loop terminates when semantic conditions are met (no tool calls, etc.)
	// Default: 100 (high enough to not interfere with normal operation)
	MaxIterations int

	// EnableExitTool adds the exit_loop tool for explicit termination.
	EnableExitTool bool

	// EnableEscalateTool adds the escalate tool for parent delegation.
	EnableEscalateTool bool

	// CompletionInstruction is appended to help the model know when to stop.
	CompletionInstruction string
}

// InstructionProvider generates instructions dynamically.
type InstructionProvider func(ctx agent.ReadonlyContext) (string, error)

// ContextProvider retrieves relevant context based on user input.
// Used for RAG context injection when IncludeContext is enabled.
// The returned string is injected into the conversation as additional context.
type ContextProvider func(ctx agent.ReadonlyContext, query string) (string, error)

// BeforeModelCallback runs before an LLM call.
// Return non-nil Response to skip the actual LLM call.
type BeforeModelCallback func(ctx agent.CallbackContext, req *model.Request) (*model.Response, error)

// AfterModelCallback runs after an LLM call.
// Return non-nil Response to replace the LLM response.
type AfterModelCallback func(ctx agent.CallbackContext, resp *model.Response, err error) (*model.Response, error)

// BeforeToolCallback runs before tool execution.
// Return non-nil result to skip actual tool execution.
type BeforeToolCallback func(ctx tool.Context, t tool.Tool, args map[string]any) (map[string]any, error)

// AfterToolCallback runs after tool execution.
// Return non-nil result to replace the tool result.
type AfterToolCallback func(ctx tool.Context, t tool.Tool, args, result map[string]any, err error) (map[string]any, error)

// IncludeContents controls conversation history handling.
type IncludeContents string

const (
	// IncludeContentsDefault includes relevant conversation history.
	IncludeContentsDefault IncludeContents = "default"

	// IncludeContentsNone only uses the current turn.
	IncludeContentsNone IncludeContents = "none"
)

// llmAgent implements agent.Agent with LLM capabilities.
type llmAgent struct {
	agent.Agent // Embedded base agent

	model           model.LLM
	instruction     string
	tools           []tool.Tool
	toolsets        []tool.Toolset
	enableStreaming bool

	instructionProvider       InstructionProvider
	globalInstruction         string
	globalInstructionProvider InstructionProvider
	generateConfig            *model.GenerateConfig

	beforeModelCallbacks []BeforeModelCallback
	afterModelCallbacks  []AfterModelCallback
	beforeToolCallbacks  []BeforeToolCallback
	afterToolCallbacks   []AfterToolCallback

	disallowTransferToParent bool
	disallowTransferToPeers  bool
	includeContents          IncludeContents
	outputKey                string
	inputSchema              map[string]any
	outputSchema             map[string]any

	// Reasoning configuration
	reasoning *ReasoningConfig

	// Working memory strategy for context window management
	workingMemory memory.WorkingMemoryStrategy

	// Context provider for RAG
	contextProvider ContextProvider

	// Processor pipeline
	pipeline *Pipeline

	// Metrics recorder for tool execution tracking
	metricsRecorder observability.Recorder
}

// New creates a new LLM-based agent.
//
// To convert an agent to a tool for agent-as-tool delegation (Pattern 2),
// use the agenttool package:
//
//	import "github.com/kadirpekel/hector/pkg/tool/agenttool"
//
//	searchAgent, _ := llmagent.New(llmagent.Config{...})
//	rootAgent, _ := llmagent.New(llmagent.Config{
//	    Tools: []tool.Tool{
//	        agenttool.New(searchAgent, nil),  // ✅ Clean factory pattern
//	    },
//	})
func New(cfg Config) (agent.Agent, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if cfg.Model == nil {
		return nil, fmt.Errorf("model is required")
	}

	// Initialize reasoning config with defaults
	reasoning := cfg.Reasoning
	if reasoning == nil {
		reasoning = &ReasoningConfig{}
	}
	if reasoning.MaxIterations == 0 {
		reasoning.MaxIterations = 100 // Safety limit, not primary control
	}

	// Initialize processor pipeline
	var pipeline *Pipeline
	if cfg.Pipeline != nil {
		pipeline = cfg.Pipeline
	} else {
		pipeline = NewPipeline()
		// Add custom processors after defaults
		for _, p := range cfg.RequestProcessors {
			pipeline.AddRequestProcessor(p)
		}
		for _, p := range cfg.ResponseProcessors {
			pipeline.AddResponseProcessor(p)
		}
	}

	a := &llmAgent{
		model:                     cfg.Model,
		instruction:               cfg.Instruction,
		tools:                     cfg.Tools,
		toolsets:                  cfg.Toolsets,
		enableStreaming:           cfg.EnableStreaming,
		instructionProvider:       cfg.InstructionProvider,
		globalInstruction:         cfg.GlobalInstruction,
		globalInstructionProvider: cfg.GlobalInstructionProvider,
		generateConfig:            cfg.GenerateConfig,
		beforeModelCallbacks:      cfg.BeforeModelCallbacks,
		afterModelCallbacks:       cfg.AfterModelCallbacks,
		beforeToolCallbacks:       cfg.BeforeToolCallbacks,
		afterToolCallbacks:        cfg.AfterToolCallbacks,
		disallowTransferToParent:  cfg.DisallowTransferToParent,
		disallowTransferToPeers:   cfg.DisallowTransferToPeers,
		includeContents:           cfg.IncludeContents,
		outputKey:                 cfg.OutputKey,
		inputSchema:               cfg.InputSchema,
		outputSchema:              cfg.OutputSchema,
		reasoning:                 reasoning,
		workingMemory:             cfg.WorkingMemory,
		contextProvider:           cfg.ContextProvider,
		pipeline:                  pipeline,
		metricsRecorder:           cfg.MetricsRecorder,
	}

	// Create base agent with our run function
	baseAgent, err := agent.New(agent.Config{
		Name:                 cfg.Name,
		Description:          cfg.Description,
		SubAgents:            cfg.SubAgents,
		BeforeAgentCallbacks: cfg.BeforeAgentCallbacks,
		Run:                  a.run,
		AfterAgentCallbacks:  cfg.AfterAgentCallbacks,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create base agent: %w", err)
	}

	a.Agent = baseAgent
	return a, nil
}

func (a *llmAgent) run(ctx agent.InvocationContext) iter.Seq2[*agent.Event, error] {
	// Use the adk-go aligned Flow for reasoning loop
	flow := NewFlow(a)
	return flow.Run(ctx)
}

// buildCompletionInstruction generates instruction text based on reasoning config.
func (a *llmAgent) buildCompletionInstruction() string {
	if a.reasoning.CompletionInstruction != "" {
		return a.reasoning.CompletionInstruction
	}

	var guidelines []string

	if a.reasoning.EnableExitTool {
		guidelines = append(guidelines, "- Call `exit_loop` when your task is complete and you have a final answer")
	}
	if a.reasoning.EnableEscalateTool {
		guidelines = append(guidelines, "- Call `escalate` if you need help, are stuck, or the task is outside your capabilities")
	}

	if len(guidelines) == 0 {
		return ""
	}

	return "## Completion Guidelines\n" + joinInstructions(guidelines)
}

// joinInstructions combines multiple instruction parts with proper spacing.
func joinInstructions(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	// Join with double newlines for clear separation
	var result string
	for i, part := range parts {
		if i > 0 {
			result += "\n\n"
		}
		result += part
	}
	return result
}

// buildMessages constructs the message history for the LLM.
// It filters events based on branch and content settings, following ADK-Go patterns.
//
// Processing pipeline:
//  1. Filter by branch
//  2. Skip partial events
//  3. Reconstruct thinking blocks with signatures
//  4. Convert foreign agent messages
//  5. Rearrange events for async function responses (adk-go pattern)
//  6. Rearrange tool calls/results for proper pairing
//  7. Filter auth events
func (a *llmAgent) buildMessages(ctx agent.InvocationContext) []*a2a.Message {
	var messages []*a2a.Message

	// Include history if configured
	if a.includeContents != IncludeContentsNone {
		session := ctx.Session()
		if session != nil {
			currentBranch := ctx.Branch()

			// Create model-aware content processor
			provider := model.ProviderUnknown
			if a.model != nil {
				provider = a.model.Provider()
			}
			processor := NewContentProcessor(a.Name(), provider)

			// Collect and filter events first
			var events []*agent.Event
			for event := range session.Events().All() {
				if event.Message == nil {
					continue
				}

				// Skip events not belonging to current branch (ADK-Go pattern)
				if !eventBelongsToBranch(currentBranch, event.Branch) {
					continue
				}

				// Skip partial/streaming events
				if event.Partial {
					continue
				}

				// Skip pending_approval tool results (HITL flow control only, not real results)
				// These are placeholder events created when a tool requires approval
				// The actual result comes after approval in executePendingApprovedTools
				isPendingApproval := false
				for _, tr := range event.ToolResults {
					if tr.Status == "pending_approval" {
						isPendingApproval = true
						break
					}
				}
				if isPendingApproval {
					slog.Debug("Skipping pending_approval event",
						"event_author", event.Author,
						"tool_results", len(event.ToolResults))
					continue
				}

				events = append(events, event)
			}

			// Apply working memory strategy to filter events for context window
			// This happens BEFORE rearranging to ensure we're working with the filtered set
			if a.workingMemory != nil {
				beforeCount := len(events)
				events = a.workingMemory.FilterEvents(events)
				if len(events) != beforeCount {
					slog.Debug("Working memory filtered events",
						"strategy", a.workingMemory.Name(),
						"before", beforeCount,
						"after", len(events))
				}
			}

			// Rearrange events for async function responses (adk-go pattern)
			// This handles long-running tools where responses arrive out of order
			events, _ = processor.RearrangeEventsForLatestFunctionResponse(events)
			events, _ = processor.RearrangeEventsForFunctionResponsesInHistory(events)

			// Convert events to messages
			for _, event := range events {
				// Reconstruct thinking blocks if present in metadata
				msg := reconstructMessageWithThinking(event)

				// Convert foreign agent messages to user context
				msg = processor.ConvertForeignAgentMessage(msg, event.Author)

				messages = append(messages, msg)
			}

			// Process messages for model-specific requirements
			// Flow creates the correct message structure for all providers:
			// - Assistant message with tool_use blocks
			// - User message with tool_result blocks
			messages = processor.Process(messages)

			// Filter out auth events
			messages = processor.FilterAuthEvents(messages)
		}
	} else {
		// IncludeContentsNone mode: only use current turn context (ADK-Go pattern)
		// The user message is already in session (runner appends it before agent runs),
		// so we read from session but only include events from the current turn.
		session := ctx.Session()
		if session != nil {
			// Find the latest user message and include from there (current turn only)
			var events []*agent.Event
			for event := range session.Events().All() {
				if event.Message != nil {
					events = append(events, event)
				}
			}

			// Find start of current turn (latest user message)
			startIdx := 0
			for i := len(events) - 1; i >= 0; i-- {
				if events[i].Author == agent.AuthorUser {
					startIdx = i
					break
				}
			}

			// Include events from current turn only
			for _, event := range events[startIdx:] {
				messages = append(messages, event.Message)
			}
		}
	}

	// NOTE: When including history, do NOT add ctx.UserContent() here!
	// The Runner already appends the user message to the session BEFORE the agent runs.
	// Adding it again would cause duplicate user messages on subsequent iterations
	// (e.g., after tool execution), which breaks the LLM's understanding of the conversation.
	// This was causing infinite tool call loops with Anthropic.
	// (ADK-Go pattern: ContentsRequestProcessor only reads from session.Events().All())

	return messages
}

// reconstructMessageWithThinking adds thinking block data to message if present in event metadata.
// CRITICAL: For Anthropic, thinking blocks must have signatures and appear before tool_use.
func reconstructMessageWithThinking(event *agent.Event) *a2a.Message {
	// If no thinking in metadata, return original message
	if event.CustomMetadata == nil {
		return event.Message
	}

	thinkingContent, hasThinking := event.CustomMetadata["thinking"].(string)
	thinkingSignature, hasSignature := event.CustomMetadata["thinking_signature"].(string)

	// Can't reconstruct without both content and signature
	if !hasThinking || !hasSignature || thinkingContent == "" || thinkingSignature == "" {
		return event.Message
	}

	// Only reconstruct for agent messages (assistant role)
	if event.Message.Role != a2a.MessageRoleAgent {
		return event.Message
	}

	// Build new parts list with thinking FIRST (Anthropic requirement)
	newParts := []a2a.Part{
		a2a.DataPart{
			Data: map[string]any{
				"type":      "thinking",
				"thinking":  thinkingContent,
				"signature": thinkingSignature,
			},
		},
	}

	// Add original parts after thinking
	newParts = append(newParts, event.Message.Parts...)

	return a2a.NewMessage(a2a.MessageRoleAgent, newParts...)
}

// eventBelongsToBranch checks if an event belongs to the given branch.
// Events belong to a branch if they match exactly or are ancestors (prefix match with dot).
// This follows ADK-Go's branch filtering pattern for multi-agent isolation.
func eventBelongsToBranch(invocationBranch, eventBranch string) bool {
	// Empty invocation branch means root level - include all events
	if invocationBranch == "" {
		return true
	}

	// Exact match
	if eventBranch == invocationBranch {
		return true
	}

	// Empty event branch means it's from root - always visible
	if eventBranch == "" {
		return true
	}

	// Check if invocation branch starts with event branch (ancestor event)
	// Use dot delimiter to avoid false matches (agent_1 vs agent_10)
	if len(invocationBranch) > len(eventBranch) &&
		invocationBranch[:len(eventBranch)] == eventBranch &&
		invocationBranch[len(eventBranch)] == '.' {
		return true
	}

	return false
}

func (a *llmAgent) collectToolDefinitions(ctx agent.InvocationContext) []tool.Definition {
	var defs []tool.Definition

	// Add control tools based on reasoning config
	controlTools := a.getControlTools()
	for _, t := range controlTools {
		defs = append(defs, tool.ToDefinition(t))
	}

	// Add static tools (both CallableTool and StreamingTool)
	for _, t := range a.tools {
		// tool.ToDefinition handles both CallableTool and StreamingTool
		defs = append(defs, tool.ToDefinition(t))
	}

	// Add toolset tools
	for _, ts := range a.toolsets {
		tools, err := ts.Tools(ctx)
		if err != nil {
			slog.Warn("Toolset failed to provide tools",
				"toolset", ts.Name(),
				"agent", a.Name(),
				"error", err)
			continue // Skip failed toolsets
		}
		for _, t := range tools {
			// tool.ToDefinition handles both CallableTool and StreamingTool
			defs = append(defs, tool.ToDefinition(t))
		}
	}

	return defs
}

// getControlTools returns control flow tools based on reasoning config.
func (a *llmAgent) getControlTools() []tool.Tool {
	var tools []tool.Tool

	if a.reasoning.EnableExitTool {
		tools = append(tools, controltool.ExitLoop())
	}

	if a.reasoning.EnableEscalateTool {
		tools = append(tools, controltool.Escalate())
	}

	return tools
}

func (a *llmAgent) findTool(ctx agent.InvocationContext, name string) tool.Tool {
	// Check control tools first
	for _, t := range a.getControlTools() {
		if t.Name() == name {
			return t
		}
	}

	// Check static tools
	for _, t := range a.tools {
		if t.Name() == name {
			return t
		}
	}

	// Check toolsets
	for _, ts := range a.toolsets {
		tools, err := ts.Tools(ctx)
		if err != nil {
			slog.Debug("Toolset error during tool lookup",
				"toolset", ts.Name(),
				"tool_name", name,
				"error", err)
			continue
		}
		for _, t := range tools {
			if t.Name() == name {
				return t
			}
		}
	}

	return nil
}

// DisallowTransferToParent returns whether parent transfer is disabled.
func (a *llmAgent) DisallowTransferToParent() bool {
	return a.disallowTransferToParent
}

// DisallowTransferToPeers returns whether peer transfer is disabled.
func (a *llmAgent) DisallowTransferToPeers() bool {
	return a.disallowTransferToPeers
}

// WorkingMemory returns the agent's working memory strategy for context window management.
// Implements memory.WorkingMemoryProvider interface.
func (a *llmAgent) WorkingMemory() memory.WorkingMemoryStrategy {
	return a.workingMemory
}

// Ensure llmAgent implements WorkingMemoryProvider.
var _ memory.WorkingMemoryProvider = (*llmAgent)(nil)
