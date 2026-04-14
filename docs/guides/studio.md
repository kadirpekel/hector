# Hector Studio

Hector Studio is the web-based UI for Hector. It provides a visual interface for designing, configuring, and testing AI agents.

> **[Try Studio online →](https://studio.gohector.dev)** — no install required. Add your server URL and start building.

[![Hector Studio](../assets/hector-studio.png)](https://studio.gohector.dev)

## Features

### 1. Visual Flow Builder
Design complex multi-agent architectures using a drag-and-drop canvas.

*   **Bi-directional Sync**: Changes in the UI write to your `hector.yaml`. Edits to the YAML file instantly reflect in the UI.
*   **Agent Types**: Drag and drop Sequential, Parallel, and Loop nodes.

### 2. Resource Management
Configure shared resources without touching YAML.

*   **LLMs**: Add and test providers (OpenAI, Anthropic, Ollama).
*   **Tools**: Search the MCP registry and install tools with one click.
*   **Guardrails**: Toggle safety policies on/off.

### 3. Integrated Chat
Test your agents in the browser.

*   **Streaming**: Watch tokens arrive in real-time.
*   **Trace View**: See exactly which tools were called and what data was passed.
*   **Markdown Support**: Full rendering of tables, code blocks, and images.

## Getting Started

### Try Online (Quickest)

Visit **[studio.gohector.dev](https://studio.gohector.dev)** and add your server URL. No install or build step needed.

### Embedded in Hector (Easiest)

Release builds of Hector include Studio bundled directly in the binary:

```bash
hector serve
```

Open `http://localhost:8080/` in your browser. Studio automatically detects it's running in embedded mode and connects to the host server. No manual configuration needed.

!!! note
    Development builds (plain `go build`) don't embed UI assets.
    You'll see a fallback page at `/` with instructions on how to set it up.

### Build with Embedded UI

To build Hector with Studio embedded:

```bash
make build-release
```

This requires **Node.js** (for `npm`). It builds Studio from the `studio/` directory, copies the assets into `ui/dist/`, and produces a single `hector` executable with the UI embedded via Go's `//go:embed`.

!!! info "Build Pipeline"
    The monorepo build pipeline:
    
    1. `studio/`: React + Vite app, built with `npm run build` → `studio/dist/`
    2. `ui/dist/`: Staging directory, assets copied here by the Makefile
    3. `ui/embed.go`: Uses `//go:embed all:dist` to bake assets into the Go binary
    4. At runtime, `ui.Handler()` serves embedded assets or shows a fallback page

### Development Mode

Run Studio's dev server with hot-reload for UI development:

```bash
cd studio
npm install
npm run dev
```

Open `http://localhost:5173`, then add your Hector server URL to connect.

## Connecting to Servers

Studio can connect to any Hector server, local or remote.

1.  Click the **Server Selector** (top-left).
2.  Click **Add Server**.
3.  Enter the server URL (e.g., `http://localhost:8080` or `https://agents.acme.com`).
4.  Authenticate if the server requires it.

Chat, configuration, and observability features all work against any connected server.

## Cloud Integration (Optional)

When cloud features are enabled, Studio can provision and manage cloud-hosted Hector instances:

1.  Click the **Server Selector**.
2.  Select **Hector Cloud**.
3.  Enter your license key (obtained from the cloud signup page).
4.  Studio provisions a managed instance and connects automatically.

Cloud features are only available when the `VITE_HECTOR_CLOUD_URL` environment variable is set at build time. See the [Studio README](https://github.com/verikod/hector/tree/main/studio) for configuration details.

## Troubleshooting

**UI shows a fallback page at `/`**

This means the binary was built from source without embedded UI assets. Options:

- Use a release binary: `go install github.com/verikod/hector/cmd/hector@latest`
- Run `make build-release` to build a binary with UI embedded (requires Node.js)
- Run Studio dev server with `npm run dev` in the `studio/` directory
- Try online at [studio.gohector.dev](https://studio.gohector.dev)

**Studio can't connect to the server**

When using the dev server (port 5173), add your server URL via **Server Selector → Add Server**. Ensure the Hector server is running and accessible from your browser.

**Assets look broken or outdated**

Hard-refresh the browser (`Cmd+Shift+R` / `Ctrl+Shift+R`) to clear cached assets.

## Next Steps

*   [Try Studio online](https://studio.gohector.dev)
*   [Quick Start Guide](../getting-started/quick-start.md)
*   [Configuration Reference](../reference/configuration.md)
