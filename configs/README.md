# Configuration Presets

This directory contains specialized configuration presets for different use cases.

**Note**: The default `hector.yaml` in the root directory is a general-purpose configuration suitable for most users. These configs show specialized presets for specific advanced use cases.

## Philosophy

The default `hector.yaml` is **already optimized for general-purpose use** with safe defaults (Tier 1). These presets show **specialized configurations** for specific advanced use cases.

## Available Presets

### 1. Research Pipeline Workflow (`research-pipeline-workflow.yaml`) - 🟢 Tier 1: Safe

**Purpose**: Demonstrates **automated multi-agent orchestration** for complex workflows

**⚡ Key Feature**: Shows Hector's **Team + Workflow system** with DAG execution, dependency management, and automatic context sharing

**Features**:
- **3 Specialized Agents** (auto-orchestrated):
  - 🔍 Researcher (GPT-4o, temp 0.3): Information gathering
  - 📊 Analyst (Claude Sonnet, temp 0.5): Data analysis
  - ✍️  Writer (GPT-4o, temp 0.7): Report synthesis
- **DAG Workflow Execution**:
  - Dependency-based execution (analyst waits for researcher)
  - Context sharing via variables (`${research_data}`)
  - Automatic progress tracking and streaming
  - Error recovery with retries
- **Single Command**: All agents execute automatically

**Use Cases**:
- Complex multi-step automation
- Specialized role-based processing
- Data pipeline orchestration
- Report generation workflows

**Usage**:
```bash
# Automatic orchestration - single command
hector --config configs/research-pipeline-workflow.yaml --workflow research_pipeline
> "Research AI agent frameworks and market adoption in 2024"

# Hector automatically:
# 1. Executes researcher agent
# 2. Passes output to analyst (waits for completion)
# 3. Combines both outputs for writer
# 4. Returns final report

# Manual single-agent mode (for testing):
hector --config configs/research-pipeline-workflow.yaml --agent researcher
> "Research AI agent frameworks"
```

### 2. Coding Assistant (`coding.yaml`) - 🟡 Tier 2: Developer Mode

**Purpose**: Complete Cursor/Claude-like experience for professional development

**⚠️  Security Notice**: This configuration includes **file editing tools** and aggressive tool usage. Only use in trusted environments with proper backups.

**Features**:
- **LLM**: Claude Sonnet 3.7 (Anthropic)
- **Experience**: Full Cursor/Claude capabilities with maximum context windows
- **Tools**: Complete developer toolkit:
  - 🔧 `execute_command` (git, go, npm, docker, etc.)
  - ✏️  `file_writer` (create/overwrite files)
  - 📝 `search_replace` (precise editing)
  - ✅ `todo_write` (task management)
- **Semantic Search**: Enabled with Qdrant + Ollama for deep codebase understanding
- **Prompt**: Optimized system prompt matching Cursor's pair programming behavior
- **Temperature**: 0.1 (precise, deterministic)
- **Max Iterations**: 10 (allows complex multi-step tasks)
- **Max Tokens**: 16,000 (large context)

**Use Cases**:
- Professional pair programming (Cursor-like workflow)
- Complex multi-file refactoring
- Code generation with context awareness
- Bug fixing and debugging
- Architecture changes and migrations
- Code review and improvement

**Usage**:
```bash
# Interactive mode (shorthand)
hector coding

# Single query
echo "Refactor the auth module to use JWT tokens" | hector coding

# Or use explicit path
hector --config configs/coding.yaml

# Prerequisites: 
# - ANTHROPIC_API_KEY (required)
# - Qdrant + Ollama (optional, for semantic search)
```


## Quick Start

### For Most Users (Default)

Just use the default configuration:
```bash
# Set your API key
export OPENAI_API_KEY="your-key"  # or ANTHROPIC_API_KEY

# Run with default config (uses hector.yaml)
hector

# Or explicitly specify it
hector --config hector.yaml
```

### For Specialized Use Cases

1. **Choose your preset**:
   - Coding tasks (Developer Mode) → `configs/coding.yaml`
   - Full Cursor experience → `configs/cursor.yaml`

2. **Set up environment variables**:
   ```bash
   export ANTHROPIC_API_KEY="your-key"  # For Claude-based configs
   export OPENAI_API_KEY="your-key"     # For GPT-based configs
   ```

3. **Run Hector**:
   ```bash
   # Short form (recommended)
   hector coding
   hector cursor
   
   # Long form
   hector --config configs/coding.yaml
   ```

## Customization

Each example can be customized by:

1. **Changing the LLM**: Modify the `llms` section
2. **Adjusting prompts**: Edit `prompt_slots` for fine-tuning
3. **Adding/removing tools**: Update the `tools` section
4. **Enabling features**: Toggle `include_context`, `show_debug_info`, etc.

## Configuration Reference

For detailed configuration options, see: [CONFIGURATION.md](../CONFIGURATION.md)

## Comparison

| Feature | Default (Root) | Research Pipeline | Coding |
|---------|----------------|-------------------|--------|
| **LLM** | OpenAI GPT-4o | Mixed (GPT+Claude) | Claude 3.7 |
| **Security Tier** | 🟢 Safe | 🟢 Safe | 🟡 Developer |
| **File Editing** | ❌ No | ❌ No | ✅ Yes |
| **Semantic Search** | ❌ Optional | ❌ No | ✅ Yes |
| **Workflow Mode** | ❌ Single Agent | ✅ DAG (3 agents) | ❌ Single |
| **Orchestration** | Manual | **Automatic** | Manual |
| **Context Sharing** | ❌ No | ✅ Yes (`${vars}`) | ❌ No |
| **Temperature** | 0.7 (balanced) | 0.3-0.7 (per role) | 0.1 (precise) |
| **Max Tokens** | 8,000 | 8,000-16,000 | 16,000 |
| **Max Iterations** | 5 | Varies | 10 |
| **Use Case** | General purpose | **Automated workflows** | **Pro development** |

## Tips

- **Getting started**: Use the default `hector.yaml`
  - ✅ Zero-config ready with safe defaults
  - ✅ General-purpose for most use cases
  - ✅ No external dependencies required
  
- **For multi-agent workflows**: See `configs/multi-agent-workflow.yaml`
  - 🟢 Safe (Tier 1)
  - ✅ Shows agent coordination patterns
  - 💡 Starting point for complex automation
  
- **For development**: Use `hector coding`
  - 🟡 Full developer capabilities (Tier 2)
  - ✅ Cursor/Claude-like experience
  - ✅ Semantic search for codebase understanding
  - ⚠️  Only in trusted environments
  
- **For production**: Start with default, customize via `prompt_slots`

