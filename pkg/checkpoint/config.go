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

package checkpoint

import (
	"fmt"
	"time"
)

// Strategy determines when checkpoints are created.
type Strategy string

const (
	// StrategyEvent - Checkpoint on specific events (tool approval, errors).
	StrategyEvent Strategy = "event"

	// StrategyInterval - Checkpoint every N iterations.
	StrategyInterval Strategy = "interval"

	// StrategyHybrid - Both event and interval checkpointing.
	StrategyHybrid Strategy = "hybrid"
)

// Config configures checkpoint behavior.
//
// Example YAML configuration:
//
//	checkpoint:
//	  enabled: true
//	  strategy: hybrid
//	  interval: 5
//	  after_tools: true
//	  before_llm: false
//	  recovery:
//	    auto_resume: true
//	    auto_resume_hitl: false
//	    timeout: 3600
type Config struct {
	// Enabled enables checkpointing.
	// Default: false
	Enabled *bool `yaml:"enabled,omitempty"`

	// Strategy determines when checkpoints are created.
	// Values: "event", "interval", "hybrid"
	// Default: "event"
	Strategy Strategy `yaml:"strategy,omitempty"`

	// Interval specifies checkpoint frequency (every N iterations).
	// Only used when Strategy is "interval" or "hybrid".
	// Default: 0 (disabled)
	Interval int `yaml:"interval,omitempty"`

	// AfterTools checkpoints after tool executions complete.
	// Default: false
	AfterTools *bool `yaml:"after_tools,omitempty"`

	// BeforeLLM checkpoints before LLM API calls.
	// Default: false
	BeforeLLM *bool `yaml:"before_llm,omitempty"`

	// Recovery configures checkpoint recovery behavior.
	Recovery *RecoveryConfig `yaml:"recovery,omitempty"`
}

// RecoveryConfig configures checkpoint recovery behavior.
type RecoveryConfig struct {
	// AutoResume enables automatic recovery on startup.
	// Default: false
	AutoResume *bool `yaml:"auto_resume,omitempty"`

	// AutoResumeHITL enables automatic recovery for INPUT_REQUIRED tasks.
	// When false, INPUT_REQUIRED tasks wait for explicit user action.
	// Default: false
	AutoResumeHITL *bool `yaml:"auto_resume_hitl,omitempty"`

	// Timeout is the maximum age (in seconds) for a checkpoint to be recoverable.
	// Checkpoints older than this are considered expired and marked as FAILED.
	// Default: 3600 (1 hour)
	Timeout int `yaml:"timeout,omitempty"`
}

// SetDefaults applies default values.
func (c *Config) SetDefaults() {
	if c.Enabled == nil {
		enabled := false
		c.Enabled = &enabled
	}
	if c.Strategy == "" {
		c.Strategy = StrategyEvent
	}
	if c.AfterTools == nil {
		afterTools := false
		c.AfterTools = &afterTools
	}
	if c.BeforeLLM == nil {
		beforeLLM := false
		c.BeforeLLM = &beforeLLM
	}
	if c.Recovery == nil {
		c.Recovery = &RecoveryConfig{}
	}
	c.Recovery.SetDefaults()
}

// SetDefaults applies default values for RecoveryConfig.
func (c *RecoveryConfig) SetDefaults() {
	if c.AutoResume == nil {
		autoResume := false
		c.AutoResume = &autoResume
	}
	if c.AutoResumeHITL == nil {
		autoResumeHITL := false
		c.AutoResumeHITL = &autoResumeHITL
	}
	if c.Timeout == 0 {
		c.Timeout = 3600 // 1 hour
	}
}

// Validate checks the configuration.
func (c *Config) Validate() error {
	if c.Strategy != "" &&
		c.Strategy != StrategyEvent &&
		c.Strategy != StrategyInterval &&
		c.Strategy != StrategyHybrid {
		return fmt.Errorf("invalid checkpoint strategy '%s' (valid: event, interval, hybrid)", c.Strategy)
	}
	if c.Interval < 0 {
		return fmt.Errorf("checkpoint interval must be non-negative")
	}
	if c.Recovery != nil {
		if err := c.Recovery.Validate(); err != nil {
			return fmt.Errorf("recovery config: %w", err)
		}
	}
	return nil
}

// Validate checks the RecoveryConfig.
func (c *RecoveryConfig) Validate() error {
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative")
	}
	return nil
}

// IsEnabled returns whether checkpointing is enabled.
func (c *Config) IsEnabled() bool {
	return c != nil && c.Enabled != nil && *c.Enabled
}

// ShouldCheckpointAfterTools returns whether to checkpoint after tool execution.
func (c *Config) ShouldCheckpointAfterTools() bool {
	return c.IsEnabled() && c.AfterTools != nil && *c.AfterTools
}

// ShouldCheckpointBeforeLLM returns whether to checkpoint before LLM calls.
func (c *Config) ShouldCheckpointBeforeLLM() bool {
	return c.IsEnabled() && c.BeforeLLM != nil && *c.BeforeLLM
}

// ShouldCheckpointInterval returns whether interval checkpointing is enabled.
func (c *Config) ShouldCheckpointInterval() bool {
	return c.IsEnabled() &&
		(c.Strategy == StrategyInterval || c.Strategy == StrategyHybrid) &&
		c.Interval > 0
}

// ShouldCheckpointAtIteration returns whether to checkpoint at the given iteration.
func (c *Config) ShouldCheckpointAtIteration(iteration int) bool {
	if !c.ShouldCheckpointInterval() {
		return false
	}
	return iteration > 0 && iteration%c.Interval == 0
}

// GetRecoveryTimeout returns the recovery timeout as a duration.
func (c *Config) GetRecoveryTimeout() time.Duration {
	if c == nil || c.Recovery == nil || c.Recovery.Timeout <= 0 {
		return time.Hour // Default 1 hour
	}
	return time.Duration(c.Recovery.Timeout) * time.Second
}

// ShouldAutoResume returns whether to auto-resume on startup.
func (c *Config) ShouldAutoResume() bool {
	return c.IsEnabled() && c.Recovery != nil && c.Recovery.AutoResume != nil && *c.Recovery.AutoResume
}

// ShouldAutoResumeHITL returns whether to auto-resume INPUT_REQUIRED tasks.
func (c *Config) ShouldAutoResumeHITL() bool {
	return c.IsEnabled() && c.Recovery != nil && c.Recovery.AutoResumeHITL != nil && *c.Recovery.AutoResumeHITL
}
