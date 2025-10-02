package workflow

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kadirpekel/hector/config"
)

// ============================================================================
// COMPREHENSIVE WORKFLOW TESTS WITH BENCHMARKS
// ============================================================================

// TestDAGWorkflowBehavior verifies DAG executor behaves correctly
func TestDAGWorkflowBehavior(t *testing.T) {
	tests := []struct {
		name       string
		numAgents  int
		input      string
		wantEvents []WorkflowEventType
	}{
		{
			name:      "Single Agent",
			numAgents: 1,
			input:     "test single agent",
			wantEvents: []WorkflowEventType{
				EventProgress,
				EventAgentStart,
				EventAgentOutput,
				EventAgentComplete,
				EventWorkflowEnd,
			},
		},
		{
			name:      "Two Agents",
			numAgents: 2,
			input:     "test two agents",
			wantEvents: []WorkflowEventType{
				EventProgress,
				EventAgentStart,
				EventAgentComplete,
				EventProgress,
				EventAgentStart,
				EventAgentComplete,
				EventWorkflowEnd,
			},
		},
		{
			name:      "Five Agents",
			numAgents: 5,
			input:     "test five agents",
			wantEvents: []WorkflowEventType{
				EventProgress,
				EventWorkflowEnd,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create agents
			agents := make([]string, tt.numAgents)
			for i := 0; i < tt.numAgents; i++ {
				agents[i] = fmt.Sprintf("agent-%d", i+1)
			}

			workflowConfig := config.WorkflowConfig{
				Name:   tt.name,
				Mode:   config.ExecutionModeDAG,
				Agents: agents,
			}

			executor := NewDAGExecutor(workflowConfig)
			mockServices := &MockAgentServices{agents: agents}

			request := &WorkflowRequest{
				Workflow:      &workflowConfig,
				AgentServices: mockServices,
				Input:         tt.input,
				Context: WorkflowContext{
					Variables: make(map[string]string),
					Metadata:  make(map[string]string),
					Artifacts: make(map[string]Artifact),
				},
			}

			ctx := context.Background()
			eventCh, err := executor.ExecuteStreaming(ctx, request)
			if err != nil {
				t.Fatalf("Failed to start streaming: %v", err)
			}

			// Collect all events
			var receivedEvents []WorkflowEventType
			eventCount := make(map[WorkflowEventType]int)
			startTime := time.Now()

			for event := range eventCh {
				receivedEvents = append(receivedEvents, event.EventType)
				eventCount[event.EventType]++
			}

			duration := time.Since(startTime)

			// Verify we got all expected event types
			for _, expectedType := range tt.wantEvents {
				if eventCount[expectedType] == 0 {
					t.Errorf("Expected at least one %s event, got 0", expectedType)
				}
			}

			// Verify correct number of agents executed
			if eventCount[EventAgentStart] != tt.numAgents {
				t.Errorf("Expected %d agent start events, got %d", tt.numAgents, eventCount[EventAgentStart])
			}

			if eventCount[EventAgentComplete] != tt.numAgents {
				t.Errorf("Expected %d agent complete events, got %d", tt.numAgents, eventCount[EventAgentComplete])
			}

			// Verify workflow ended
			if eventCount[EventWorkflowEnd] != 1 {
				t.Errorf("Expected 1 workflow end event, got %d", eventCount[EventWorkflowEnd])
			}

			t.Logf("✅ %s: %d agents, %d events, %.2fs",
				tt.name, tt.numAgents, len(receivedEvents), duration.Seconds())
		})
	}
}

// TestAutonomousWorkflowBehavior verifies Autonomous executor behaves correctly
func TestAutonomousWorkflowBehavior(t *testing.T) {
	tests := []struct {
		name           string
		numAgents      int
		input          string
		expectedMinRun int // Minimum agents expected to run
	}{
		{
			name:           "Single Agent",
			numAgents:      1,
			input:          "test autonomous single",
			expectedMinRun: 1,
		},
		{
			name:           "Three Agents",
			numAgents:      3,
			input:          "test autonomous three",
			expectedMinRun: 3,
		},
		{
			name:           "Ten Agents",
			numAgents:      10,
			input:          "test autonomous ten",
			expectedMinRun: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create agents
			agents := make([]string, tt.numAgents)
			for i := 0; i < tt.numAgents; i++ {
				agents[i] = fmt.Sprintf("agent-%d", i+1)
			}

			workflowConfig := config.WorkflowConfig{
				Name:   tt.name,
				Mode:   config.ExecutionModeAutonomous,
				Agents: agents,
			}

			executor := NewAutonomousExecutor(workflowConfig)
			mockServices := &MockAgentServices{agents: agents}

			request := &WorkflowRequest{
				Workflow:      &workflowConfig,
				AgentServices: mockServices,
				Input:         tt.input,
				Context: WorkflowContext{
					Variables: make(map[string]string),
					Metadata:  make(map[string]string),
					Artifacts: make(map[string]Artifact),
				},
			}

			ctx := context.Background()
			eventCh, err := executor.ExecuteStreaming(ctx, request)
			if err != nil {
				t.Fatalf("Failed to start streaming: %v", err)
			}

			// Collect events
			eventCount := make(map[WorkflowEventType]int)
			startTime := time.Now()

			for event := range eventCh {
				eventCount[event.EventType]++
			}

			duration := time.Since(startTime)

			// Verify minimum agents ran
			agentsRun := eventCount[EventAgentStart]
			if agentsRun < tt.expectedMinRun {
				t.Errorf("Expected at least %d agents to run, got %d", tt.expectedMinRun, agentsRun)
			}

			// Verify workflow completed
			if eventCount[EventWorkflowEnd] != 1 {
				t.Errorf("Expected 1 workflow end event, got %d", eventCount[EventWorkflowEnd])
			}

			// Verify planning event
			if eventCount[EventProgress] < 1 {
				t.Errorf("Expected at least 1 progress event (planning), got %d", eventCount[EventProgress])
			}

			t.Logf("✅ %s: %d agents, %d ran, %.2fs",
				tt.name, tt.numAgents, agentsRun, duration.Seconds())
		})
	}
}

// TestEventOrdering verifies events come in correct order
func TestEventOrdering(t *testing.T) {
	workflowConfig := config.WorkflowConfig{
		Name:   "ordering-test",
		Mode:   config.ExecutionModeDAG,
		Agents: []string{"agent-1", "agent-2"},
	}

	executor := NewDAGExecutor(workflowConfig)
	mockServices := &MockAgentServices{agents: []string{"agent-1", "agent-2"}}

	request := &WorkflowRequest{
		Workflow:      &workflowConfig,
		AgentServices: mockServices,
		Input:         "test ordering",
		Context: WorkflowContext{
			Variables: make(map[string]string),
			Metadata:  make(map[string]string),
			Artifacts: make(map[string]Artifact),
		},
	}

	ctx := context.Background()
	eventCh, err := executor.ExecuteStreaming(ctx, request)
	if err != nil {
		t.Fatalf("Failed to start streaming: %v", err)
	}

	var events []WorkflowEvent
	for event := range eventCh {
		events = append(events, event)
	}

	// Verify agent-1 completes before agent-2 starts (sequential execution)
	agent1CompleteIdx := -1
	agent2StartIdx := -1

	for i, event := range events {
		if event.EventType == EventAgentComplete && event.AgentName == "agent-1" {
			agent1CompleteIdx = i
		}
		if event.EventType == EventAgentStart && event.AgentName == "agent-2" {
			agent2StartIdx = i
		}
	}

	if agent1CompleteIdx < 0 || agent2StartIdx < 0 {
		t.Fatal("Missing agent completion or start events")
	}

	if agent1CompleteIdx >= agent2StartIdx {
		t.Errorf("Expected agent-1 to complete before agent-2 starts, but agent-1 completed at index %d and agent-2 started at %d",
			agent1CompleteIdx, agent2StartIdx)
	}

	t.Logf("✅ Event ordering correct: agent-1 complete (idx %d) → agent-2 start (idx %d)",
		agent1CompleteIdx, agent2StartIdx)
}

// TestProgressAccuracy verifies progress tracking is accurate
func TestProgressAccuracy(t *testing.T) {
	numAgents := 5
	agents := make([]string, numAgents)
	for i := 0; i < numAgents; i++ {
		agents[i] = fmt.Sprintf("agent-%d", i+1)
	}

	workflowConfig := config.WorkflowConfig{
		Name:   "progress-test",
		Mode:   config.ExecutionModeDAG,
		Agents: agents,
	}

	executor := NewDAGExecutor(workflowConfig)
	mockServices := &MockAgentServices{agents: agents}

	request := &WorkflowRequest{
		Workflow:      &workflowConfig,
		AgentServices: mockServices,
		Input:         "test progress",
		Context: WorkflowContext{
			Variables: make(map[string]string),
			Metadata:  make(map[string]string),
			Artifacts: make(map[string]Artifact),
		},
	}

	ctx := context.Background()
	eventCh, err := executor.ExecuteStreaming(ctx, request)
	if err != nil {
		t.Fatalf("Failed to start streaming: %v", err)
	}

	var progressEvents []WorkflowProgress
	for event := range eventCh {
		if event.Progress != nil {
			progressEvents = append(progressEvents, *event.Progress)
		}
	}

	// Verify we got progress events
	if len(progressEvents) != numAgents {
		t.Errorf("Expected %d progress events, got %d", numAgents, len(progressEvents))
	}

	// Verify progress increases monotonically
	for i, progress := range progressEvents {
		expectedCompleted := i
		expectedPercent := float64(i) / float64(numAgents) * 100

		if progress.CompletedSteps != expectedCompleted {
			t.Errorf("Progress event %d: expected %d completed steps, got %d",
				i, expectedCompleted, progress.CompletedSteps)
		}

		if progress.TotalSteps != numAgents {
			t.Errorf("Progress event %d: expected %d total steps, got %d",
				i, numAgents, progress.TotalSteps)
		}

		if progress.PercentComplete != expectedPercent {
			t.Errorf("Progress event %d: expected %.1f%% complete, got %.1f%%",
				i, expectedPercent, progress.PercentComplete)
		}
	}

	t.Logf("✅ Progress tracking accurate across %d steps", numAgents)
}

// ============================================================================
// BENCHMARKS
// ============================================================================

// BenchmarkDAGExecutor benchmarks DAG workflow execution
func BenchmarkDAGExecutor(b *testing.B) {
	benchmarks := []struct {
		name      string
		numAgents int
	}{
		{"1-Agent", 1},
		{"2-Agents", 2},
		{"5-Agents", 5},
		{"10-Agents", 10},
		{"20-Agents", 20},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Create agents
			agents := make([]string, bm.numAgents)
			for i := 0; i < bm.numAgents; i++ {
				agents[i] = fmt.Sprintf("agent-%d", i+1)
			}

			workflowConfig := config.WorkflowConfig{
				Name:   "benchmark-dag",
				Mode:   config.ExecutionModeDAG,
				Agents: agents,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				executor := NewDAGExecutor(workflowConfig)
				mockServices := &MockAgentServices{agents: agents}

				request := &WorkflowRequest{
					Workflow:      &workflowConfig,
					AgentServices: mockServices,
					Input:         fmt.Sprintf("benchmark iteration %d", i),
					Context: WorkflowContext{
						Variables: make(map[string]string),
						Metadata:  make(map[string]string),
						Artifacts: make(map[string]Artifact),
					},
				}

				ctx := context.Background()
				eventCh, err := executor.ExecuteStreaming(ctx, request)
				if err != nil {
					b.Fatalf("Failed to start streaming: %v", err)
				}

				// Consume all events
				eventCount := 0
				for range eventCh {
					eventCount++
				}
			}
		})
	}
}

// BenchmarkAutonomousExecutor benchmarks Autonomous workflow execution
func BenchmarkAutonomousExecutor(b *testing.B) {
	benchmarks := []struct {
		name      string
		numAgents int
	}{
		{"1-Agent", 1},
		{"2-Agents", 2},
		{"5-Agents", 5},
		{"10-Agents", 10},
		{"20-Agents", 20},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Create agents
			agents := make([]string, bm.numAgents)
			for i := 0; i < bm.numAgents; i++ {
				agents[i] = fmt.Sprintf("agent-%d", i+1)
			}

			workflowConfig := config.WorkflowConfig{
				Name:   "benchmark-autonomous",
				Mode:   config.ExecutionModeAutonomous,
				Agents: agents,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				executor := NewAutonomousExecutor(workflowConfig)
				mockServices := &MockAgentServices{agents: agents}

				request := &WorkflowRequest{
					Workflow:      &workflowConfig,
					AgentServices: mockServices,
					Input:         fmt.Sprintf("benchmark iteration %d", i),
					Context: WorkflowContext{
						Variables: make(map[string]string),
						Metadata:  make(map[string]string),
						Artifacts: make(map[string]Artifact),
					},
				}

				ctx := context.Background()
				eventCh, err := executor.ExecuteStreaming(ctx, request)
				if err != nil {
					b.Fatalf("Failed to start streaming: %v", err)
				}

				// Consume all events
				for range eventCh {
				}
			}
		})
	}
}

// BenchmarkEventThroughput measures event processing throughput
func BenchmarkEventThroughput(b *testing.B) {
	workflowConfig := config.WorkflowConfig{
		Name:   "throughput-test",
		Mode:   config.ExecutionModeDAG,
		Agents: []string{"agent-1"},
	}

	executor := NewDAGExecutor(workflowConfig)
	mockServices := &MockAgentServices{agents: []string{"agent-1"}}

	b.ResetTimer()
	totalEvents := 0

	for i := 0; i < b.N; i++ {
		request := &WorkflowRequest{
			Workflow:      &workflowConfig,
			AgentServices: mockServices,
			Input:         "throughput test",
			Context: WorkflowContext{
				Variables: make(map[string]string),
				Metadata:  make(map[string]string),
				Artifacts: make(map[string]Artifact),
			},
		}

		ctx := context.Background()
		eventCh, _ := executor.ExecuteStreaming(ctx, request)

		for range eventCh {
			totalEvents++
		}
	}

	b.ReportMetric(float64(totalEvents)/b.Elapsed().Seconds(), "events/sec")
}

// TestWorkflowResponses verifies response content is correct
func TestWorkflowResponses(t *testing.T) {
	workflowConfig := config.WorkflowConfig{
		Name:   "response-test",
		Mode:   config.ExecutionModeDAG,
		Agents: []string{"agent-1", "agent-2"},
	}

	executor := NewDAGExecutor(workflowConfig)
	mockServices := &MockAgentServices{agents: []string{"agent-1", "agent-2"}}

	request := &WorkflowRequest{
		Workflow:      &workflowConfig,
		AgentServices: mockServices,
		Input:         "test responses",
		Context: WorkflowContext{
			Variables: make(map[string]string),
			Metadata:  make(map[string]string),
			Artifacts: make(map[string]Artifact),
		},
	}

	ctx := context.Background()
	eventCh, err := executor.ExecuteStreaming(ctx, request)
	if err != nil {
		t.Fatalf("Failed to start streaming: %v", err)
	}

	// Collect output
	var fullOutput strings.Builder
	agentOutputs := make(map[string]string)
	currentAgent := ""

	for event := range eventCh {
		switch event.EventType {
		case EventAgentStart:
			currentAgent = event.AgentName
		case EventAgentOutput:
			fullOutput.WriteString(event.Content)
			agentOutputs[currentAgent] += event.Content
		case EventWorkflowEnd:
			// Verify final output contains results
			if !strings.Contains(event.Content, "Mock result from") {
				t.Errorf("Expected workflow end to contain results, got: %s", event.Content)
			}
		}
	}

	// Verify both agents produced output
	if len(agentOutputs["agent-1"]) == 0 {
		t.Error("agent-1 produced no output")
	}
	if len(agentOutputs["agent-2"]) == 0 {
		t.Error("agent-2 produced no output")
	}

	// Verify output contains expected messages
	expectedMessages := []string{
		"Analyzing input...",
		"Processing data...",
		"Generating response...",
	}

	for _, msg := range expectedMessages {
		if !strings.Contains(fullOutput.String(), msg) {
			t.Errorf("Expected output to contain '%s'", msg)
		}
	}

	t.Logf("✅ Response validation passed")
	t.Logf("   agent-1 output: %d bytes", len(agentOutputs["agent-1"]))
	t.Logf("   agent-2 output: %d bytes", len(agentOutputs["agent-2"]))
	t.Logf("   total output: %d bytes", fullOutput.Len())
}
