package tools

import (
	"context"
	"fmt"
	"strings"
)

// StreamingOrchestrator handles the orchestration of streaming tool execution.
// It manages channel creation, goroutine coordination, content accumulation,
// and incremental result emission, separating streaming concerns from the agent core.
type StreamingOrchestrator struct {
	bufferSize int
}

// NewStreamingOrchestrator creates a new streaming orchestrator with the specified buffer size.
func NewStreamingOrchestrator(bufferSize int) *StreamingOrchestrator {
	return &StreamingOrchestrator{
		bufferSize: bufferSize,
	}
}

// ChunkEmitter is a function that emits accumulated content chunks.
// It should return an error if emission fails (e.g., context cancelled).
type ChunkEmitter func(accumulatedContent string) error

// Execute orchestrates the execution of a streaming tool.
// It handles channel management, goroutine coordination, content accumulation,
// and incremental chunk emission via the provided emitter function.
//
// Example usage:
//
//	orchestrator := NewStreamingOrchestrator(10)
//	result, err := orchestrator.Execute(ctx, streamingTool, toolCall.Args, func(content string) error {
//		return safeSendPart(ctx, outputCh, protocol.CreateToolResultPart(&protocol.ToolResult{
//			ToolCallID: toolCall.ID,
//			Content:    content,
//		}))
//	})
func (o *StreamingOrchestrator) Execute(
	ctx context.Context,
	tool StreamingTool,
	args map[string]interface{},
	emitChunk ChunkEmitter,
) (ToolResult, error) {
	resultCh := make(chan string, o.bufferSize)
	var accumulated strings.Builder
	var finalResult ToolResult
	var execErr error

	// Start streaming execution in a goroutine
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		finalResult, execErr = tool.ExecuteStreaming(ctx, args, resultCh)
	}()

	// Stream incremental tool result parts
	streaming := true
	for streaming {
		select {
		case chunk, ok := <-resultCh:
			if !ok {
				streaming = false
				break
			}
			if chunk != "" {
				accumulated.WriteString(chunk)
				// Emit incremental tool result part
				if err := emitChunk(accumulated.String()); err != nil {
					return ToolResult{}, err
				}
			}
		case <-ctx.Done():
			return ToolResult{}, ctx.Err()
		case <-doneCh:
			streaming = false
		}
	}

	// Wait for final result
	<-doneCh

	// Use final result content if accumulated content is empty, otherwise prefer accumulated
	content := accumulated.String()
	if content == "" && finalResult.Content != "" {
		content = finalResult.Content
	}

	errorStr := ""
	if execErr != nil {
		if content == "" {
			content = fmt.Sprintf("Error: %v", execErr)
		}
		errorStr = execErr.Error()
	} else if finalResult.Error != "" {
		errorStr = finalResult.Error
	}

	return ToolResult{
		Content:  content,
		Error:    errorStr,
		Metadata: finalResult.Metadata,
		Success:  execErr == nil,
		ToolName: finalResult.ToolName,
	}, execErr
}
