---
layout: default
title: Installation
nav_order: 1
parent: Getting Started
description: "Essential installation methods for Hector"
---

# Installing Hector

Essential installation methods for the Hector AI Agent Platform.

## Prerequisites

- **Go 1.24+** (for building from source)
- **API Key** from your LLM provider (OpenAI, Anthropic, or Gemini)

---

## Installation Methods

### 1. Binary Releases

Pre-built binaries are available for all major platforms:

#### Linux (x64)
```bash
curl -L https://github.com/kadirpekel/hector/releases/latest/download/hector-linux-amd64 -o hector
chmod +x hector
sudo mv hector /usr/local/bin/
```

#### macOS (Intel)
```bash
curl -L https://github.com/kadirpekel/hector/releases/latest/download/hector-darwin-amd64 -o hector
chmod +x hector
sudo mv hector /usr/local/bin/
```

#### macOS (Apple Silicon)
```bash
curl -L https://github.com/kadirpekel/hector/releases/latest/download/hector-darwin-arm64 -o hector
chmod +x hector
sudo mv hector /usr/local/bin/
```

#### Windows (x64)
```powershell
curl -L https://github.com/kadirpekel/hector/releases/latest/download/hector-windows-amd64.exe -o hector.exe
# Move to a directory in your PATH or run from current directory
```

### 2. Go Install

Install directly using Go package manager:

```bash
go install github.com/kadirpekel/hector/cmd/hector@latest
```

**Note:** Requires Go 1.24 or later. The binary will be installed to `$GOPATH/bin` (typically `~/go/bin`).

### 3. Build from Source

Clone and build the project:

```bash
# Clone repository
git clone https://github.com/kadirpekel/hector.git
cd hector

# Build binary
go build -o hector ./cmd/hector

# Optionally, install to system
sudo mv hector /usr/local/bin/
```

### 4. Docker

Run Hector in a container:

```bash
# Pull the image
docker pull kadirpekel/hector:latest

# Run with configuration
docker run -d \
  --name hector \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -e OPENAI_API_KEY="your-api-key" \
  kadirpekel/hector:latest \
  serve --config /app/config.yaml
```

#### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'
services:
  hector:
    image: kadirpekel/hector:latest
    ports:
      - "8080:8080"
      - "8081:8081"  # REST gateway
      - "8082:8082"  # JSON-RPC
    volumes:
      - ./config.yaml:/app/config.yaml
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    command: serve --config /app/config.yaml
    restart: unless-stopped
```

Run:
```bash
docker-compose up -d
```

---

## Verify Installation

After installation, verify everything works:

```bash
# Check version
hector version

# Run zero-config mode (no config file needed)
export OPENAI_API_KEY="sk-..."
hector call "Hello, world!"  # Agent name optional in zero-config!
```

---

## Environment Setup

### API Keys

Set up your LLM provider API keys:

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."

# Google Gemini
export GEMINI_API_KEY="AI..."
```

**Using .env file:**
```bash
cat > .env << EOF
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
GEMINI_API_KEY=AI...
EOF
```

### Configuration File

Create a basic configuration file:

```bash
cat > config.yaml << EOF
agents:
  assistant:
    name: "My Assistant"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are a helpful assistant who provides clear,
        concise answers.

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "\${OPENAI_API_KEY}"
    temperature: 0.7
EOF
```

Test your configuration:

```bash
# Start server
hector serve --config config.yaml

# In another terminal, test the agent
hector list
hector call "Explain quantum computing"  # Agent name optional in zero-config!
```

