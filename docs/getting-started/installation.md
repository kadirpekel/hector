# Installation

Install Hector using one of the methods below. Release binaries include the [Studio UI](../guides/studio.md) built in.

## Install Script (Recommended)

=== "macOS / Linux"

    ```bash
    curl -fsSL https://gohector.dev/install.sh | sh
    ```

=== "Windows (PowerShell)"

    ```powershell
    irm https://gohector.dev/install.ps1 | iex
    ```

=== "Homebrew"

    ```bash
    brew install verikod/tap/hector
    ```

To install a specific version:

```bash
curl -fsSL https://gohector.dev/install.sh | sh -s -- --version v1.2.3
```

## Download Binary

Download a prebuilt binary from the [latest release](https://github.com/verikod/hector/releases/latest).

1. Download the archive for your OS and architecture (e.g. `hector_X.Y.Z_darwin_arm64.tar.gz`)
2. Extract it
3. Move the `hector` binary to a directory in your `PATH`

```bash
tar xzf hector_*_darwin_arm64.tar.gz
sudo mv hector /usr/local/bin/
```

On Windows, extract the `.zip` file and place `hector.exe` in a directory on your `PATH` (e.g. `C:\Users\you\.hector\bin`).

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
    Building from source with `go build` produces a headless binary (no Studio UI). Use `make build-release` instead to include Studio (requires Node.js). See the [Studio Guide](../guides/studio.md) for details.

## Go Install (Headless)

If you only need the API server without the Studio UI:

```bash
go install github.com/verikod/hector/cmd/hector@latest
```

*Requires Go 1.24+. This produces a headless binary — the web UI will not be available.*

## Verify Installation

```bash
hector version
```

## Next Steps

- [Quick Start](quick-start.md) - Get an agent running in 5 minutes
- [Hector Studio](../guides/studio.md) - Visual UI for designing and testing agents
- [CLI Reference](../reference/cli.md) - All CLI commands and flags
- [Configuration Reference](../reference/configuration.md) - YAML configuration format
