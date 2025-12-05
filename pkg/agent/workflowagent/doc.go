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

// Package workflowagent provides workflow agents for orchestrating multi-agent flows.
//
// This package provides three types of workflow agents aligned with adk-go:
//
// # SequentialAgent
//
// Runs sub-agents once, in the order they are listed:
//
//	agent, _ := workflowagent.NewSequential(workflowagent.SequentialConfig{
//	    Name:        "pipeline",
//	    Description: "Processes data through multiple stages",
//	    SubAgents:   []agent.Agent{stage1, stage2, stage3},
//	})
//
// # ParallelAgent
//
// Runs sub-agents simultaneously in parallel:
//
//	agent, _ := workflowagent.NewParallel(workflowagent.ParallelConfig{
//	    Name:        "voters",
//	    Description: "Gets multiple perspectives simultaneously",
//	    SubAgents:   []agent.Agent{voter1, voter2, voter3},
//	})
//
// # LoopAgent
//
// Runs sub-agents repeatedly for N iterations or until escalation:
//
//	agent, _ := workflowagent.NewLoop(workflowagent.LoopConfig{
//	    Name:          "refiner",
//	    Description:   "Iteratively refines output",
//	    SubAgents:     []agent.Agent{reviewer, improver},
//	    MaxIterations: 3,
//	})
package workflowagent
