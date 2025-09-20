# Hector Configuration Examples

This directory contains 3 essential configuration examples that provide maximum coverage of Hector's capabilities with minimal maintenance overhead.

## Examples

### 1. `basic.yaml` - Basic Setup with Tools
**Purpose**: Minimal configuration with tool support
**Features**: 
- LLM configuration (OpenAI or Ollama)
- MCP tool integration
- Basic document model
- Tool-enabled reasoning
**Use Case**: Quick start with tool capabilities, learning, minimal resource usage

```bash
hector --config examples/basic.yaml
```

### 2. `advanced.yaml` - Advanced Features
**Purpose**: Demonstrates complex reasoning and tool integration
**Features**: 
- Multi-step reasoning workflows
- Nested agent hierarchies
- MCP tool integration
- Advanced context management
**Use Case**: Complex problem solving, enterprise applications

```bash
hector --config examples/advanced.yaml
```

### 3. `document-ingestion.yaml` - Document Automation
**Purpose**: Automated document sync with PDF, Word, and text file support
**Features**: 
- Single universal document model
- PDF text extraction
- Word (.docx) text extraction
- Text and Markdown support
- Pattern matching and exclusions
**Use Case**: Knowledge base automation, document management

```bash
hector --config examples/document-ingestion.yaml
```

## Coverage Matrix

| Feature | basic.yaml | advanced.yaml | document-ingestion.yaml |
|---------|------------|---------------|------------------------|
| **Minimal Setup** | ✅ | ❌ | ❌ |
| **Local Ollama** | ✅ | ❌ | ❌ |
| **MCP Tools** | ✅ | ✅ | ❌ |
| **Multi-step Reasoning** | ❌ | ✅ | ❌ |
| **Nested Agents** | ❌ | ✅ | ❌ |
| **Document Ingestion** | ❌ | ❌ | ✅ |
| **PDF Support** | ❌ | ❌ | ✅ |
| **Word Support** | ❌ | ❌ | ✅ |
| **Text Support** | ❌ | ❌ | ✅ |

## Setup Instructions

1. **Replace API Keys**: 
   - Update `YOUR_OPENAI_API_KEY_HERE` with your actual OpenAI API key
   - Update `YOUR_COMPOSIO_API_KEY_HERE` with your Composio API key (for tool support)
2. **Adjust Paths**: Modify source paths in `document-ingestion.yaml` to match your environment
3. **Choose Provider**: In `basic.yaml`, uncomment either OpenAI or Ollama configuration
4. **Prerequisites**: Ensure Ollama and Qdrant are running for local examples

## Prerequisites

```bash
# For Ollama examples
ollama serve

# For Qdrant
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

## CLI Commands

Once running, try these commands:

```bash
# Document ingestion
/list-models
/sync-model documents
/search "your query" documents

# General
/help
/tools
/quit
```
