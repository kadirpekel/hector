# Installation

Install Hector using one of the methods below.

## Go Install (Recommended)

Simplest way if you have Go installed.

```bash
go install github.com/verikod/hector/cmd/hector@latest
```

*Requires Go 1.24+*

## Docker

Run Hector as a container.

**GitHub Container Registry (Recommended):**
```bash
docker pull ghcr.io/verikod/hector:latest
```

**Docker Hub:**
```bash
docker pull verikod/hector:latest
```

### Run with Docker

```bash
docker run -p 8080:8080 \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  ghcr.io/verikod/hector:latest serve
```

## From Source

Build the binary from source for maximum control.

```bash
git clone https://github.com/verikod/hector.git
cd hector
go build -o hector ./cmd/hector
```

!!! note "Web UI in source builds"
    Building from source with `go build` produces a headless binary. The root `/` will show a fallback page. Use `make build-release` instead to include Studio (requires Node.js). See the [Studio Guide](../guides/studio.md) for details.

## Verify Installation

```bash
hector version
```

## Next Steps

- [Quick Start](quick-start.md) - Get an agent running in 5 minutes
- [Hector Studio](../guides/studio.md) - Visual UI for designing and testing agents
- [CLI Reference](../reference/cli.md) - All CLI commands and flags
- [Configuration Reference](../reference/configuration.md) - YAML configuration format
