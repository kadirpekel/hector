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

package workflowagent

import (
	"fmt"
	"iter"

	"golang.org/x/sync/errgroup"

	"github.com/kadirpekel/hector/pkg/agent"
)

// ParallelConfig defines the configuration for a ParallelAgent.
type ParallelConfig struct {
	// Name is the agent name.
	Name string

	// Description describes what the agent does.
	Description string

	// SubAgents are the agents to run in parallel.
	SubAgents []agent.Agent
}

// NewParallel creates a ParallelAgent.
//
// ParallelAgent runs its sub-agents in parallel in an isolated manner.
// All sub-agents receive the same input and run simultaneously.
//
// This is beneficial for scenarios requiring multiple perspectives or
// attempts on a single task, such as:
//   - Running different algorithms simultaneously
//   - Generating multiple responses for review by an evaluation agent
//   - Getting diverse perspectives on a problem
//
// Example:
//
//	voter1, _ := llmagent.New(llmagent.Config{Name: "voter1", ...})
//	voter2, _ := llmagent.New(llmagent.Config{Name: "voter2", ...})
//	voter3, _ := llmagent.New(llmagent.Config{Name: "voter3", ...})
//
//	voters, _ := workflowagent.NewParallel(workflowagent.ParallelConfig{
//	    Name:        "voters",
//	    Description: "Gets multiple perspectives simultaneously",
//	    SubAgents:   []agent.Agent{voter1, voter2, voter3},
//	})
func NewParallel(cfg ParallelConfig) (agent.Agent, error) {
	return agent.New(agent.Config{
		Name:        cfg.Name,
		Description: cfg.Description,
		SubAgents:   cfg.SubAgents,
		Run: func(ctx agent.InvocationContext) iter.Seq2[*agent.Event, error] {
			return runParallel(ctx)
		},
	})
}

// result holds an event or error from a sub-agent.
type result struct {
	event *agent.Event
	err   error
}

func runParallel(ctx agent.InvocationContext) iter.Seq2[*agent.Event, error] {
	return func(yield func(*agent.Event, error) bool) {
		var (
			errGroup, errGroupCtx = errgroup.WithContext(ctx)
			doneChan              = make(chan bool)
			resultsChan           = make(chan result)
		)

		curAgent := ctx.Agent()

		// Start all sub-agents in parallel
		for _, sa := range curAgent.SubAgents() {
			subAgent := sa // Capture for goroutine
			branch := fmt.Sprintf("%s/%s", curAgent.Name(), subAgent.Name())
			if ctx.Branch() != "" {
				branch = fmt.Sprintf("%s/%s", ctx.Branch(), branch)
			}

			errGroup.Go(func() error {
				subCtx := agent.NewInvocationContext(errGroupCtx, agent.InvocationContextParams{
					Agent:       subAgent,
					Session:     ctx.Session(),
					Artifacts:   ctx.Artifacts(),
					Memory:      ctx.Memory(),
					UserContent: ctx.UserContent(),
					RunConfig:   ctx.RunConfig(),
					Branch:      branch,
				})

				if err := runSubAgent(subCtx, subAgent, resultsChan, doneChan); err != nil {
					return fmt.Errorf("failed to run sub-agent %q: %w", subAgent.Name(), err)
				}
				return nil
			})
		}

		// Close results channel when all goroutines complete
		go func() {
			_ = errGroup.Wait()
			close(resultsChan)
		}()

		// Yield results as they come in
		defer close(doneChan)
		for res := range resultsChan {
			if !yield(res.event, res.err) {
				break
			}
		}
	}
}

func runSubAgent(ctx agent.InvocationContext, ag agent.Agent, results chan<- result, done <-chan bool) error {
	for event, err := range ag.Run(ctx) {
		select {
		case <-done:
			return nil
		case <-ctx.Done():
			select {
			case <-done:
			case results <- result{err: ctx.Err()}:
			}
			return ctx.Err()
		case results <- result{event: event, err: err}:
			if err != nil {
				return err
			}
		}
	}
	return nil
}
