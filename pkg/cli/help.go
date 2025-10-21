package cli

import "fmt"

// ShowHelp displays the complete CLI help message
func ShowHelp() {
	fmt.Print(`
Hector - AI Agent Platform

USAGE:
  hector <command> [options]

COMMANDS:
  serve              Start A2A server to host agents
  list               List available agents
  info <agent>       Get agent information
  call [agent] "..."  Execute a task on an agent (agent required in client mode)
  chat [agent]       Start interactive chat (agent required in client mode)
  task <action> <agent> <task-id>  Manage tasks (actions: get, cancel)
  help               Show this help message
  version            Show version information

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
THREE MODES OF OPERATION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Hector operates in three distinct modes based on your command and flags:

1️⃣  SERVER MODE - Host agents for multiple clients
   Trigger: 'serve' command
   Use when: Production deployments, multi-agent systems, team access
   Supports: --config AND zero-config flags

2️⃣  CLIENT MODE - Connect to remote Hector server
   Trigger: --server flag
   Use when: Accessing remote/production servers, team collaboration
   Supports: ONLY --server, --token, --stream
   ⚠️  --config and zero-config flags NOT supported (server has its own config)

3️⃣  LOCAL MODE - Run agents in-process (no server)
   Trigger: No --server flag
   Use when: Quick tasks, local development, scripts, experimentation
   Supports: --config AND zero-config flags

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔧 SERVER MODE - Start persistent server
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  hector serve [options]
    --config FILE            Configuration file (required)
    --port PORT              gRPC server port (default: 8080, overrides config)
    --host HOST              Server host (overrides config)
    --a2a-base-url URL       A2A base URL for discovery (overrides config)
    --debug                  Enable debug output
    
  Zero-Config Options (when --config is not provided):
    --provider PROVIDER      LLM provider: openai|anthropic|gemini (auto-detected)
    --api-key KEY            API key (or set env var, see below)
    --model MODEL            Model name (provider-specific defaults)
    --base-url URL           API base URL (provider-specific defaults)
    --tools                  Enable all local tools
    --mcp-url URL            MCP server URL (supports auth: https://user:pass@host)
    --docs FOLDER            Document store folder (RAG)
    --embedder-model MODEL   Embedder model (default: nomic-embed-text)
    --vectordb URL           Vector DB (default: http://localhost:6334)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🌐 CLIENT MODE - Connect to remote server
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  hector list [options]
    --server URL     Server URL (triggers client mode)
    --token TOKEN    Authentication token

  hector info <agent> [options]
    --server URL     Server URL (triggers client mode)
    --token TOKEN    Authentication token

  hector call [agent] "prompt" [options]
    --server URL     Server URL (triggers client mode)
    --token TOKEN    Authentication token
    --stream BOOL    Enable streaming (default: true)

  hector chat [agent] [options]
    --server URL     Server URL (triggers client mode)
    --token TOKEN    Authentication token

  ⚠️  Important: --config, --provider, --model, --tools, --api-key flags are NOT 
      supported in client mode. The remote server uses its own configuration.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

💻 LOCAL MODE - In-process execution (no server)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Same commands as Client mode, but WITHOUT --server flag:

  hector list [--config FILE]
  hector info <agent> [--config FILE]
  hector call "prompt" [--config FILE] [zero-config options]  # Local mode
  hector call <agent> "prompt" [--config FILE]               # Local mode with config
  hector chat [--config FILE] [zero-config options]          # Local mode  
  hector chat <agent> [--config FILE]                       # Local mode with config

  With Config File:
    --config FILE    Configuration file path

  Zero-Config Options (for call and chat):
    --provider PROVIDER    LLM provider: openai|anthropic|gemini (auto-detected)
    --api-key KEY          API key (or set env var, see below)
    --base-url URL         API base URL (provider-specific defaults)
    --model MODEL          Model name (provider-specific defaults)
    --tools                Enable local tools
    --mcp-url URL          MCP server URL (supports auth: https://user:pass@host)
    --docs FOLDER          Document store folder (enables RAG)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

EXAMPLES:
  
  Server Mode - Host agents:
    $ hector serve                                    # Use config file
    $ hector serve --model gpt-4o --tools             # Zero-config mode
    $ hector serve --config prod.yaml --port 9090     # Custom config & port
  
  Client Mode - Connect to remote server:
    $ hector list --server http://remote:8080         # List remote agents
    $ hector call assistant "task" --server URL       # Execute on remote
    $ hector chat coder --server URL --token abc123   # Chat with auth
  
  Local Mode - In-process execution:
    $ hector list                                     # List from local config
    $ hector call "task"                              # Zero-config (fastest!)
    $ hector call "task" --config my.yaml            # Use specific config
    $ hector call "task" --model gpt-4o              # Override model
    $ hector call "task" --docs ./documents          # Enable RAG with documents
    $ hector chat --tools                             # Enable tools
    $ hector chat --docs ./documents                  # Enable RAG with documents
    $ hector call assistant "task" --config my.yaml  # Config mode with agent
    $ hector chat assistant --config my.yaml         # Config mode with agent

  Mode Selection Examples:
    # Same command, different modes:
    $ hector call "task"                    # Local mode (local, zero-config)
    $ hector call agent "task"              # Local mode (local, config file)
    $ hector call agent "task" --server URL # Client mode (remote)

ENVIRONMENT VARIABLES:
  API Keys (for zero-config mode - auto-detected by provider):
    OPENAI_API_KEY       OpenAI (GPT) models
    ANTHROPIC_API_KEY    Anthropic (Claude) models
    GEMINI_API_KEY       Google Gemini models
  
  MCP Configuration:
    MCP_URL              MCP server URL (supports auth: https://user:pass@host)

MODE DETECTION:
  • If you use 'serve' command → Server mode
  • If you use --server flag → Client mode
  • Otherwise → Local mode

For more information: https://github.com/kadirpekel/hector
`)
}
