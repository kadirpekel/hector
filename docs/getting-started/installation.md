---
title: Installation
description: Install Hector on macOS, Linux, Windows, or Docker
---

# Installation

Get Hector installed and ready to use.

## Prerequisites

- **API Key** from an LLM provider ([OpenAI](https://platform.openai.com/api-keys), [Anthropic](https://console.anthropic.com/), or [Gemini](https://aistudio.google.com/app/apikey))
- **Go 1.24+** (only if building from source)

---

## Choose Your Installation Method

### Binary Releases (Recommended)

Pre-built binaries for all major platforms:

=== "macOS (Apple Silicon)"
    ```bash
    curl -L https://github.com/kadirpekel/hector/releases/latest/download/hector-darwin-arm64 -o hector
    chmod +x hector
    sudo mv hector /usr/local/bin/
    ```

=== "macOS (Intel)"
    ```bash
    curl -L https://github.com/kadirpekel/hector/releases/latest/download/hector-darwin-amd64 -o hector
    chmod +x hector
    sudo mv hector /usr/local/bin/
    ```

=== "Linux (x64)"
    ```bash
    curl -L https://github.com/kadirpekel/hector/releases/latest/download/hector-linux-amd64 -o hector
    chmod +x hector
    sudo mv hector /usr/local/bin/
    ```

=== "Windows (x64)"
    ```powershell
    curl -L https://github.com/kadirpekel/hector/releases/latest/download/hector-windows-amd64.exe -o hector.exe
    # Move to a directory in your PATH
    ```

### Go Install

Install using Go package manager:

```bash
go install github.com/kadirpekel/hector/cmd/hector@latest
```

!!! info
    Requires Go 1.24+. Binary installs to `$GOPATH/bin` (typically `~/go/bin`).

### Build from Source

Clone and build:

```bash
git clone https://github.com/kadirpekel/hector.git
cd hector
go build -o hector ./cmd/hector
sudo mv hector /usr/local/bin/
```

### Docker

Run in a container:

```bash
docker pull kadirpekel/hector:latest

docker run -d \
  --name hector \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -e OPENAI_API_KEY="your-api-key" \
  kadirpekel/hector:latest \
  serve --config /app/config.yaml
```

**Docker Compose:**

```yaml
version: '3.8'
services:
  hector:
    image: kadirpekel/hector:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    command: serve --config /app/config.yaml
    restart: unless-stopped
```

```bash
docker-compose up -d
```

---

## Verify Installation

Check that Hector is installed:

```bash
hector version
```

You should see output like:
```
Hector version 0.x.x
```

---

## Set Up API Key

Hector needs an API key to communicate with LLM providers.

**Option 1: Environment Variable (Recommended)**

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."

# Google Gemini
export GEMINI_API_KEY="AI..."
```

**Option 2: .env File**

```bash
cat > .env << EOF
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
GEMINI_API_KEY=AI...
EOF
```

---

## Quick Test

Test Hector with zero-config mode (no configuration file needed):

```bash
export OPENAI_API_KEY="sk-..."
hector call "What is the capital of France?"
```

You should see a response from the agent!

---

## Next Steps

- **[Quick Start](quick-start.md)** - Run your first agent and explore basic features
- **[Core Concepts](../core-concepts/overview.md)** - Learn how Hector agents work
- **[Configuration Reference](../reference/configuration.md)** - Full configuration options

---

## Platform-Specific Notes

### macOS

If you see a security warning when running Hector:

```bash
# Allow the binary to run
xattr -d com.apple.quarantine /usr/local/bin/hector
```

### Linux

Ensure `/usr/local/bin` is in your PATH:

```bash
echo $PATH | grep -q "/usr/local/bin" || echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### Windows

Add Hector to your PATH in PowerShell:

```powershell
$env:Path += ";C:\path\to\hector"
# Make it permanent
[System.Environment]::SetEnvironmentVariable("Path", $env:Path, [System.EnvironmentVariableTarget]::User)
```

