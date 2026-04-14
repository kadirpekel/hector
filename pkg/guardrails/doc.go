// Package guardrails provides composable safety controls for Hector agents.
//
// Guardrails can be applied at multiple interception points:
//   - Input: Validate and sanitize user input before agent processing
//   - Output: Filter and redact LLM responses before returning to users
//   - Tool: Authorize and validate tool calls before execution
//
// # Architecture
//
// Guardrails integrate with Hector's existing callback system:
//   - InputGuardrail -> BeforeAgentCallback
//   - OutputGuardrail -> AfterModelCallback
//   - ToolGuardrail -> BeforeToolCallback
//
// # Usage
//
// Create guardrails and chain them together:
//
//	chain := guardrails.NewInputChain(
//	    input.NewLengthValidator(10, 10000),
//	    input.NewInjectionDetector(),
//	    input.NewSanitizer(),
//	)
//
//	agent, _ := builder.NewAgent("secure-agent").
//	    WithLLM(llm).
//	    WithInputGuardrails(chain.Guardrails()...).
//	    Build()
//
// # Configuration
//
// Guardrails can be configured programmatically or via YAML:
//
//	config, _ := guardrails.LoadConfig("guardrails.yaml")
//	chain := config.BuildInputChain()
package guardrails
