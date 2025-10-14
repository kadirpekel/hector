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
hector call assistant "Hello, world!"
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
hector call assistant "Explain quantum computing"
```

---

## Platform-Specific Notes

### macOS

**Security warning:** If you get "cannot be opened because it is from an unidentified developer":

```bash
# Remove quarantine attribute
sudo xattr -rd com.apple.quarantine ./hector
```

**Apple Silicon (M1/M2/M3):** Use the ARM64 binary for best performance.

### Linux

**System dependencies** (if building from source):

```bash
# Ubuntu/Debian
sudo apt update && sudo apt install -y git curl build-essential

# RHEL/CentOS/Fedora
sudo dnf install -y git curl gcc make
```

**Run as systemd service:**

```bash
sudo tee /etc/systemd/system/hector.service << EOF
[Unit]
Description=Hector AI Agent Platform
After=network.target

[Service]
Type=simple
User=hector
WorkingDirectory=/opt/hector
ExecStart=/usr/local/bin/hector serve --config /opt/hector/config.yaml
Restart=always
RestartSec=10
Environment="OPENAI_API_KEY=your-key"

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable hector
sudo systemctl start hector
sudo systemctl status hector
```

### Windows

**PowerShell execution policy:**

```powershell
# If you get execution policy errors
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

**Add to PATH:**

```powershell
# Add hector.exe directory to PATH
$env:PATH += ";C:\path\to\hector"
```

---

## Development Setup

For contributors and developers:

```bash
# Clone repository
git clone https://github.com/kadirpekel/hector.git
cd hector

# Install dependencies
go mod download

# Run tests
make test

# Build
make build

# Run development server
./hector serve --config configs/example.yaml
```

---

## Next Steps

After installation:

1. **[Quick Start](QUICK_START.html)** - Run your first agent
2. **[Configuration Reference](CONFIGURATION.html)** - Learn all options
3. **[Building Agents](AGENTS.html)** - Core concepts and examples

**Ready to start?** → [Quick Start Guide](QUICK_START.html)
