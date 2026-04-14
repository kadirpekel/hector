// Package trigger provides scheduled and event-driven agent invocation.
//
// The scheduler component uses cron expressions to trigger agents
// on a schedule without external HTTP requests.
package trigger

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/config"
	"github.com/verikod/hector/pkg/observability"
)

// AgentInvoker is a function that invokes an agent with optional input.
// Returns the A2A task ID for tracking, or empty string if task tracking is unavailable.
// Deprecated: Use AgentEnqueuer for durable, queue-based execution.
type AgentInvoker func(ctx context.Context, agentName, input string) (taskID string, err error)

// AgentEnqueuer is a function that enqueues an agent invocation for durable execution.
// The task is processed by the worker pool with retry support.
type AgentEnqueuer func(ctx context.Context, appName, agentName, input, contextID string) error

// Scheduler manages scheduled agent invocations.
type Scheduler struct {
	cron     *cron.Cron
	enqueuer AgentEnqueuer
	appName  string
	mu       sync.Mutex
	entries  map[string]cron.EntryID // agentName -> entryID
	metrics  *observability.Metrics
}

// NewScheduler creates a new scheduler with queue-based execution.
func NewScheduler(appName string, enqueuer AgentEnqueuer, metrics *observability.Metrics) *Scheduler {
	return &Scheduler{
		cron:     cron.New(cron.WithSeconds()),
		enqueuer: enqueuer,
		appName:  appName,
		entries:  make(map[string]cron.EntryID),
		metrics:  metrics,
	}
}

// RegisterAgent registers an agent's trigger with the scheduler.
func (s *Scheduler) RegisterAgent(agentName string, ag agent.Agent, cfg *config.TriggerConfig) error {
	if cfg == nil || !cfg.IsEnabled() {
		return nil
	}

	if cfg.Type != config.TriggerTypeSchedule {
		// Ignore non-schedule triggers (e.g. webhooks are handled by HTTP server)
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Parse timezone
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone %q: %w", cfg.Timezone, err)
	}

	// Create cron schedule with timezone
	schedule, err := cron.ParseStandard(cfg.Cron)
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", cfg.Cron, err)
	}

	// Wrap schedule with timezone
	tzSchedule := &tzCronSchedule{
		schedule: schedule,
		loc:      loc,
	}

	// Capture values for closure
	input := cfg.Input
	appName := s.appName

	// Create job
	job := cron.FuncJob(func() {
		slog.Info("Trigger firing",
			"agent", agentName,
			"app", appName,
			"schedule", cfg.Cron,
			"input", input)

		s.metrics.RecordSchedulerTrigger(appName, agentName)

		// Generate unique context ID for this scheduled run
		contextID := fmt.Sprintf("schedule-%s-%d", agentName, time.Now().UnixNano())

		// Enqueue for durable execution
		if err := s.enqueuer(context.Background(), appName, agentName, input, contextID); err != nil {
			s.metrics.RecordSchedulerError(appName, agentName, "enqueue_error")
			slog.Error("Failed to enqueue scheduled task",
				"agent", agentName,
				"app", appName,
				"error", err)
		} else {
			slog.Info("Scheduled task enqueued",
				"agent", agentName,
				"app", appName,
				"context_id", contextID)
		}
	})

	// Add to cron
	entryID := s.cron.Schedule(tzSchedule, job)
	s.entries[agentName] = entryID

	slog.Info("Registered scheduled trigger",
		"agent", agentName,
		"app", appName,
		"cron", cfg.Cron,
		"timezone", cfg.Timezone)

	return nil
}

// Start begins the scheduler.
func (s *Scheduler) Start() {
	slog.Info("Starting trigger scheduler",
		"registered_agents", len(s.entries))
	s.cron.Start()
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() context.Context {
	slog.Info("Stopping trigger scheduler")
	return s.cron.Stop()
}

// tzCronSchedule wraps a Schedule with timezone awareness.
type tzCronSchedule struct {
	schedule cron.Schedule
	loc      *time.Location
}

func (s *tzCronSchedule) Next(t time.Time) time.Time {
	return s.schedule.Next(t.In(s.loc))
}
