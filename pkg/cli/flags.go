package cli

import "flag"

// zeroConfigFlags holds pointers to zero-config flag values
// This consolidates flag definitions to avoid duplication
type zeroConfigFlags struct {
	provider *string
	apiKey   *string
	baseURL  *string
	model    *string
	tools    *bool
	mcpURL   *string
	docs     *string
}

// addZeroConfigFlags adds zero-config flags to a FlagSet
// This prevents duplicating flag definitions across serve, call, and chat commands
func addZeroConfigFlags(fs *flag.FlagSet) *zeroConfigFlags {
	return &zeroConfigFlags{
		provider: fs.String("provider", "", "LLM provider: openai, anthropic, or gemini (auto-detected from API key if not set)"),
		apiKey:   fs.String("api-key", "", "LLM API key (OPENAI_API_KEY, ANTHROPIC_API_KEY, or GEMINI_API_KEY)"),
		baseURL:  fs.String("base-url", "", "LLM API base URL (provider-specific defaults if not set)"),
		model:    fs.String("model", "", "LLM model name (provider-specific defaults if not set)"),
		tools:    fs.Bool("tools", false, "Enable all local tools (file, command execution)"),
		mcpURL:   fs.String("mcp-url", "", "MCP server URL for tool integration (supports auth: https://user:pass@host)"),
		docs:     fs.String("docs", "", "Document store folder (enables RAG)"),
	}
}

// populateArgs populates CLIArgs with zero-config flag values
// This prevents duplicating flag assignments across serve, call, and chat commands
func (f *zeroConfigFlags) populateArgs(args *CLIArgs) {
	args.Provider = *f.provider
	args.APIKey = *f.apiKey
	args.BaseURL = *f.baseURL
	args.Model = *f.model
	args.Tools = *f.tools
	args.MCPURL = *f.mcpURL
	args.DocsFolder = *f.docs
}
