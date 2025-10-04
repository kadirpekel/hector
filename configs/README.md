# Configuration Presets

This directory contains specialized configuration presets for different use cases.

**Note**: The default `hector.yaml` in the root directory is a general-purpose configuration suitable for most users. These configs show specialized presets for specific advanced use cases.

## Philosophy

The default `hector.yaml` is **already optimized for general-purpose use** with safe defaults (Tier 1). These presets show **specialized configurations** for specific advanced use cases.

## Available Presets

### 1. Coding (`coding.yaml`) - 🟡 Tier 2: Developer Mode

**Purpose**: Specialized AI pair programmer for software development tasks

**⚠️  Security Notice**: This configuration includes **file editing tools** (file_writer, search_replace). Only use in trusted environments with proper backups.

**Features**:
- **LLM**: Claude Sonnet 3.7 (Anthropic)
- **Tools**: Full developer toolkit including:
  - 🔧 `execute_command` (full command set: git, go, npm, etc.)
  - ✏️  `file_writer` (create/overwrite files)
  - 📝 `search_replace` (edit files)
  - ✅ `todo_write` (task management)
- **Semantic Search**: Enabled with Qdrant + Ollama embeddings for codebase understanding
- **Prompt**: Optimized for coding tasks, file operations, and technical problem-solving
- **Temperature**: 0.1 (precise, deterministic)

**Use Cases**:
- Code generation and refactoring
- Bug fixing and debugging
- Codebase exploration and analysis
- Multi-file project creation
- Code review and improvement

**Usage**:
```bash
# Interactive mode (shorthand)
hector coding

# Single query
echo "Create a REST API in Go with /health endpoint" | hector coding

# Or use explicit path
hector --config configs/coding.yaml

# Prerequisites: Set ANTHROPIC_API_KEY, run Qdrant (optional) and Ollama (optional)
```

### 2. Cursor (`cursor.yaml`) - 🟡 Tier 2: Advanced

**Purpose**: Full Cursor/Claude-like experience with advanced features

**⚠️  Security Notice**: This configuration includes **file editing tools** and aggressive tool usage. Only use in trusted environments.

**Features**:
- **LLM**: Claude Sonnet 3.7 (Anthropic)
- **Tools**: Full toolkit including file editing
- **Semantic Search**: Enabled
- **Prompt**: Complete system prompt matching Cursor's behavior
- **Temperature**: 0.1 (precise)
- All advanced features enabled
- Maximum context and iteration limits

**Use Cases**:
- Exact Cursor/Claude replication
- Maximum capability testing
- Advanced development workflows

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

| Feature | Default (Root) | Coding Assistant | Cursor Replication |
|---------|----------------|------------------|--------------------|
| **LLM** | OpenAI GPT-4o | Claude 3.7 Sonnet | Claude 3.7 Sonnet |
| **Security Tier** | 🟢 Tier 1 (Safe) | 🟡 Tier 2 (Dev) | 🟡 Tier 2 (Advanced) |
| **File Editing** | ❌ No | ✅ Yes | ✅ Yes |
| **Semantic Search** | ❌ Optional | ✅ Yes | ✅ Yes |
| **Temperature** | 0.7 (balanced) | 0.1 (precise) | 0.1 (precise) |
| **Max Tokens** | 8,000 | 16,000 | 16,000 |
| **Tool Behavior** | Balanced | Aggressive | Aggressive |
| **Use Case** | General purpose | Software development | Full Cursor replication |

## Tips

- **Getting started**: Use the default `hector.yaml`
  - ✅ Zero-config ready with safe defaults
  - ✅ General-purpose for most use cases
  - ✅ No external dependencies required
  
- **For development**: Use `hector coding`
  - 🟡 Enables file editing (Tier 2)
  - ✅ Semantic search for codebase understanding
  - ⚠️  Only in trusted environments
  
- **For Cursor experience**: Use `hector cursor`
  - 🟡 Full capabilities (Tier 2)
  - ✅ Exact Cursor/Claude behavior
  - ⚠️  Advanced users only
  
- **For production**: Start with default, customize via `prompt_slots`

