package hector

import (
	"github.com/kadirpekel/hector/pkg/config"
)

// HITLConfigBuilder provides a fluent API for building HITL configuration
type HITLConfigBuilder struct {
	config *config.HITLConfig
}

// NewHITLConfigBuilder creates a new HITL config builder
func NewHITLConfigBuilder(cfg *config.HITLConfig) *HITLConfigBuilder {
	if cfg == nil {
		cfg = &config.HITLConfig{}
	}
	return &HITLConfigBuilder{
		config: cfg,
	}
}

// Mode sets the HITL mode ("auto", "blocking", or "async")
func (b *HITLConfigBuilder) Mode(mode string) *HITLConfigBuilder {
	if mode != "auto" && mode != "blocking" && mode != "async" {
		panic("HITL mode must be 'auto', 'blocking', or 'async'")
	}
	b.config.Mode = mode
	return b
}

// Build returns the HITL config
func (b *HITLConfigBuilder) Build() *config.HITLConfig {
	return b.config
}

// CheckpointConfigBuilder provides a fluent API for building checkpoint configuration
type CheckpointConfigBuilder struct {
	config *config.CheckpointConfig
}

// NewCheckpointConfigBuilder creates a new checkpoint config builder
func NewCheckpointConfigBuilder(cfg *config.CheckpointConfig) *CheckpointConfigBuilder {
	if cfg == nil {
		cfg = &config.CheckpointConfig{}
	}
	return &CheckpointConfigBuilder{
		config: cfg,
	}
}

// Enabled enables or disables checkpointing
func (b *CheckpointConfigBuilder) Enabled(enabled bool) *CheckpointConfigBuilder {
	b.config.Enabled = enabled
	return b
}

// Strategy sets the checkpoint strategy ("event", "interval", or "hybrid")
func (b *CheckpointConfigBuilder) Strategy(strategy string) *CheckpointConfigBuilder {
	if strategy != "event" && strategy != "interval" && strategy != "hybrid" {
		panic("checkpoint strategy must be 'event', 'interval', or 'hybrid'")
	}
	b.config.Strategy = strategy
	return b
}

// Interval creates an interval config builder
func (b *CheckpointConfigBuilder) Interval() *CheckpointIntervalConfigBuilder {
	if b.config.Interval == nil {
		b.config.Interval = &config.CheckpointIntervalConfig{}
	}
	return NewCheckpointIntervalConfigBuilder(b.config.Interval)
}

// Recovery creates a recovery config builder
func (b *CheckpointConfigBuilder) Recovery() *CheckpointRecoveryConfigBuilder {
	if b.config.Recovery == nil {
		b.config.Recovery = &config.CheckpointRecoveryConfig{}
	}
	return NewCheckpointRecoveryConfigBuilder(b.config.Recovery)
}

// Build returns the checkpoint config
func (b *CheckpointConfigBuilder) Build() *config.CheckpointConfig {
	return b.config
}

// CheckpointIntervalConfigBuilder provides a fluent API for building interval config
type CheckpointIntervalConfigBuilder struct {
	config *config.CheckpointIntervalConfig
}

// NewCheckpointIntervalConfigBuilder creates a new interval config builder
func NewCheckpointIntervalConfigBuilder(cfg *config.CheckpointIntervalConfig) *CheckpointIntervalConfigBuilder {
	if cfg == nil {
		cfg = &config.CheckpointIntervalConfig{}
	}
	return &CheckpointIntervalConfigBuilder{
		config: cfg,
	}
}

// EveryNIterations sets checkpoint interval (checkpoint every N iterations)
func (b *CheckpointIntervalConfigBuilder) EveryNIterations(n int) *CheckpointIntervalConfigBuilder {
	if n < 0 {
		panic("every_n_iterations must be non-negative")
	}
	b.config.EveryNIterations = n
	return b
}

// AfterToolCalls enables checkpointing after tool calls
func (b *CheckpointIntervalConfigBuilder) AfterToolCalls(enabled bool) *CheckpointIntervalConfigBuilder {
	b.config.AfterToolCalls = enabled
	return b
}

// BeforeLLMCalls enables checkpointing before LLM calls
func (b *CheckpointIntervalConfigBuilder) BeforeLLMCalls(enabled bool) *CheckpointIntervalConfigBuilder {
	b.config.BeforeLLMCalls = enabled
	return b
}

// Build returns the interval config
func (b *CheckpointIntervalConfigBuilder) Build() *config.CheckpointIntervalConfig {
	return b.config
}

// CheckpointRecoveryConfigBuilder provides a fluent API for building recovery config
type CheckpointRecoveryConfigBuilder struct {
	config *config.CheckpointRecoveryConfig
}

// NewCheckpointRecoveryConfigBuilder creates a new recovery config builder
func NewCheckpointRecoveryConfigBuilder(cfg *config.CheckpointRecoveryConfig) *CheckpointRecoveryConfigBuilder {
	if cfg == nil {
		cfg = &config.CheckpointRecoveryConfig{}
	}
	return &CheckpointRecoveryConfigBuilder{
		config: cfg,
	}
}

// AutoResume enables or disables auto-resume on startup
func (b *CheckpointRecoveryConfigBuilder) AutoResume(enabled bool) *CheckpointRecoveryConfigBuilder {
	b.config.AutoResume = enabled
	return b
}

// AutoResumeHITL enables or disables auto-resume for INPUT_REQUIRED tasks
func (b *CheckpointRecoveryConfigBuilder) AutoResumeHITL(enabled bool) *CheckpointRecoveryConfigBuilder {
	b.config.AutoResumeHITL = enabled
	return b
}

// ResumeTimeout sets the resume timeout in seconds
func (b *CheckpointRecoveryConfigBuilder) ResumeTimeout(seconds int) *CheckpointRecoveryConfigBuilder {
	if seconds < 0 {
		panic("resume_timeout must be non-negative")
	}
	b.config.ResumeTimeout = seconds
	return b
}

// Build returns the recovery config
func (b *CheckpointRecoveryConfigBuilder) Build() *config.CheckpointRecoveryConfig {
	return b.config
}
