// Package runneragent provides a deterministic tool execution agent.
//
// A runner agent executes its configured tools in sequence without LLM
// involvement. This enables pure automation pipelines that can be composed
// with LLM agents in workflows.
//
// Key characteristics:
//   - No LLM reasoning - tools execute deterministically in order
//   - Output piping - each tool's output becomes the next tool's input
//   - Composable - works with sequential/parallel/loop workflow agents
//   - A2A compatible - final output becomes an artifact/message
//
// Example configuration:
//
//	agents:
//	  data_fetcher:
//	    type: runner
//	    tools: [web_fetch]
//
//	  pipeline:
//	    type: sequential
//	    sub_agents: [data_fetcher, analyzer]
//
// Use cases:
//   - ETL pipelines (fetch → transform → save)
//   - CI/CD automation (bash → grep_search)
//   - Data preprocessing before LLM analysis
package runneragent
