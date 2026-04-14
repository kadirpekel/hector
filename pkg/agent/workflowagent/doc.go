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
