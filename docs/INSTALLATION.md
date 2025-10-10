---
layout: default
title: Installation
nav_order: 1
parent: Getting Started
description: "Complete installation guide with all available options"
---

# Installing Hector

Multiple installation options for different environments and use cases.

## Quick Install (Recommended)

### Prerequisites

- Go 1.25+ (for building from source)
- OpenAI API key (or other LLM provider key)

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/kadirpekel/hector.git
cd hector

# Build the binary
go build -o hector ./cmd/hector

# Verify installation
./hector version
```

### Option 2: Using Go Install

```bash
# Install directly with Go
go install github.com/kadirpekel/hector/cmd/hector@latest

# Verify installation
hector version
```

---

## Installation Options

### 1. Binary Releases (Coming Soon)

Pre-built binaries for major platforms:

```bash
# Linux (x64)
curl -L https://github.com/kadirpekel/hector/releases/latest/download/hector-linux-amd64 -o hector
chmod +x hector

# macOS (x64)
curl -L https://github.com/kadirpekel/hector/releases/latest/download/hector-darwin-amd64 -o hector
chmod +x hector

# macOS (ARM64)
curl -L https://github.com/kadirpekel/hector/releases/latest/download/hector-darwin-arm64 -o hector
chmod +x hector

# Windows (x64)
curl -L https://github.com/kadirpekel/hector/releases/latest/download/hector-windows-amd64.exe -o hector.exe
```

### 2. Docker Installation

```bash
# Pull the official image
docker pull kadirpekel/hector:latest

# Run with configuration
docker run -d \
  --name hector \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -e OPENAI_API_KEY="your-key" \
  kadirpekel/hector:latest serve --config /app/config.yaml
```

### 3. Docker Compose

Create `docker-compose.yml`:

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

Run:
```bash
docker-compose up -d
```

### 4. Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hector
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hector
  template:
    metadata:
      labels:
        app: hector
    spec:
      containers:
      - name: hector
        image: kadirpekel/hector:latest
        ports:
        - containerPort: 8080
        env:
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: hector-secrets
              key: openai-api-key
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
        command: ["./hector", "serve", "--config", "/app/config.yaml"]
      volumes:
      - name: config
        configMap:
          name: hector-config
---
apiVersion: v1
kind: Service
metadata:
  name: hector-service
spec:
  selector:
    app: hector
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
  type: LoadBalancer
```

### 5. Package Managers (Coming Soon)

**Homebrew (macOS/Linux):**
```bash
brew install kadirpekel/tap/hector
```

**Chocolatey (Windows):**
```bash
choco install hector
```

**APT (Ubuntu/Debian):**
```bash
curl -fsSL https://packages.hector.ai/gpg | sudo apt-key add -
echo "deb https://packages.hector.ai/apt stable main" | sudo tee /etc/apt/sources.list.d/hector.list
sudo apt update && sudo apt install hector
```

---

## Environment Setup

### 1. API Keys

Set up your LLM provider API keys:

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Anthropic Claude
export ANTHROPIC_API_KEY="sk-ant-..."

# Google Gemini
export GEMINI_API_KEY="AI..."

# Or create .env file
echo "OPENAI_API_KEY=sk-..." > .env
echo "ANTHROPIC_API_KEY=sk-ant-..." >> .env
```

### 2. Configuration File

Create your first configuration file:

```bash
# Create config directory
mkdir -p ~/.hector

# Create basic config
cat > ~/.hector/config.yaml << EOF
global:
  a2a_server:
    enabled: true
    host: "0.0.0.0"
    port: 8080

agents:
  assistant:
    name: "My Assistant"
    description: "A helpful AI assistant"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are a helpful assistant who provides clear,
        concise answers and explanations.

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "\${OPENAI_API_KEY}"
    temperature: 0.7
EOF
```

### 3. Verify Installation

```bash
# Check version
hector version

# Test configuration
hector validate --config ~/.hector/config.yaml

# Start server
hector serve --config ~/.hector/config.yaml

# Test in another terminal
hector list
hector call assistant "Hello, world!"
```

---

## Development Setup

### For Contributing to Hector

```bash
# Clone repository
git clone https://github.com/kadirpekel/hector.git
cd hector

# Install development dependencies
go mod download

# Run tests
make test

# Build development version
make build

# Run with hot reload (requires air)
go install github.com/cosmtrek/air@latest
air
```

### IDE Setup

**VS Code:**
```json
{
  "go.toolsManagement.checkForUpdates": "local",
  "go.useLanguageServer": true,
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint"
}
```

**GoLand:**
- Enable Go modules support
- Set GOROOT to Go 1.25+
- Configure run configurations for `cmd/hector`

---

## Platform-Specific Notes

### macOS

**Security Note:** If you get "cannot be opened because it is from an unidentified developer":
```bash
# Allow the binary
sudo xattr -rd com.apple.quarantine ./hector
```

**Apple Silicon (M1/M2):**
- Use ARM64 binaries for best performance
- Go cross-compilation works seamlessly

### Linux

**Dependencies:**
```bash
# Ubuntu/Debian
sudo apt update && sudo apt install -y git curl build-essential

# RHEL/CentOS/Fedora
sudo yum install -y git curl gcc make
# or
sudo dnf install -y git curl gcc make
```

**Systemd Service:**
```bash
# Create service file
sudo tee /etc/systemd/system/hector.service << EOF
[Unit]
Description=Hector AI Agent Platform
After=network.target

[Service]
Type=simple
User=hector
WorkingDirectory=/opt/hector
ExecStart=/opt/hector/hector serve --config /opt/hector/config.yaml
Restart=always
RestartSec=10
Environment=OPENAI_API_KEY=your-key-here

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl enable hector
sudo systemctl start hector
```

### Windows

**PowerShell Execution Policy:**
```powershell
# If you get execution policy errors
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

**Windows Service (using NSSM):**
```batch
# Download NSSM and install Hector as service
nssm install Hector "C:\path\to\hector.exe"
nssm set Hector Parameters "serve --config C:\path\to\config.yaml"
nssm set Hector AppDirectory "C:\path\to\hector"
nssm start Hector
```

---

## Cloud Deployment

### AWS EC2

```bash
# Launch EC2 instance (Ubuntu 22.04 LTS)
# SSH into instance

# Install Hector
curl -L https://github.com/kadirpekel/hector/releases/latest/download/hector-linux-amd64 -o hector
chmod +x hector
sudo mv hector /usr/local/bin/

# Create config and service
sudo useradd -r -s /bin/false hector
sudo mkdir -p /etc/hector
sudo chown hector:hector /etc/hector

# Deploy config and start service
# (see systemd service example above)
```

### Google Cloud Run

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o hector ./cmd/hector

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/hector .
COPY config.yaml .
CMD ["./hector", "serve", "--config", "config.yaml"]
```

Deploy:
```bash
gcloud run deploy hector \
  --image gcr.io/your-project/hector \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --port 8080
```

### Azure Container Instances

```bash
az container create \
  --resource-group myResourceGroup \
  --name hector \
  --image kadirpekel/hector:latest \
  --dns-name-label hector-unique \
  --ports 8080 \
  --environment-variables OPENAI_API_KEY=your-key \
  --command-line "./hector serve --config /app/config.yaml"
```

---

## Troubleshooting Installation

### Common Issues

**"Command not found: hector"**
- Ensure binary is in PATH: `export PATH=$PATH:$(pwd)`
- Or move to system location: `sudo mv hector /usr/local/bin/`

**"Permission denied"**
- Make binary executable: `chmod +x hector`
- Check file ownership: `ls -la hector`

**"Go version too old"**
- Update Go: `go version` should show 1.21+
- Install latest Go from https://golang.org/dl/

**"Module not found"**
- Run `go mod download` in project directory
- Ensure you're in the correct directory with `go.mod`

**"Port already in use"**
- Change port in config: `port: 8081`
- Kill existing process: `sudo lsof -ti:8080 | xargs kill -9`

### Getting Help

- **Documentation:** [https://gohector.dev](https://gohector.dev)
- **GitHub Issues:** [Report bugs](https://github.com/kadirpekel/hector/issues)
- **Discussions:** [Community support](https://github.com/kadirpekel/hector/discussions)
- **Discord:** [Real-time chat](https://discord.gg/hector) (coming soon)

---

## Next Steps

After installation:

1. **[Quick Start](QUICK_START)** - Run your first agent in 5 minutes
2. **[Building Agents](AGENTS)** - Learn core concepts
3. **[Configuration](CONFIGURATION)** - Explore all options
4. **[Tutorials](tutorials/)** - Hands-on learning

**Ready to build your first agent?** → [Quick Start Guide](QUICK_START)
