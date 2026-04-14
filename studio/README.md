# Hector Studio

The web-based UI for [Hector](https://github.com/verikod/hector), included in the monorepo under `studio/`.

**[Live →](https://studio.gohector.dev)** · **[Docs](https://gohector.dev/guides/studio/)** · **[Hector](https://github.com/verikod/hector)**

[![Hector Studio](https://gohector.dev/assets/hector-studio.png)](https://studio.gohector.dev)

## Features

- **Visual Flow Builder**: Drag-and-drop canvas for multi-agent architectures with bi-directional YAML sync
- **Integrated Chat**: Streaming responses, trace view, markdown rendering
- **Resource Management**: Configure LLMs, tools, and guardrails without touching YAML
- **Server Connectivity**: Connect to any Hector server (local or remote)
- **Cloud Integration**: Optional managed cloud instances (requires license key)

## Quick Start

From the repository root:

```bash
cd studio
npm install
npm run dev
```

Open `http://localhost:5173` and add a Hector server URL to connect.

## Embedded Mode

Hector Studio is embedded in [Hector](https://github.com/verikod/hector) release builds. When you run `hector serve`, the UI is available at `http://localhost:8080/`.

Development builds (plain `go build`) produce a headless binary. To build with Studio embedded:

```bash
# From the hector repo root (requires Node.js)
make build-release
```

This builds Studio, copies assets into `ui/dist/`, and compiles them into the Go binary via `//go:embed`.

## Environment Variables

| Variable | Description | Required |
|---|---|---|
| `VITE_HECTOR_CLOUD_URL` | Cloud control plane URL. Enables cloud features when set. | No |
| `VITE_CLOUD_SIGNUP_URL` | URL to purchase a license key (shown in cloud auth modal). | No |

See [.env.example](.env.example) for reference.

## Scripts

| Command | Description |
|---|---|
| `npm run dev` | Start dev server with hot reload |
| `npm run build` | Production build to `dist/` |
| `npm run preview` | Preview production build locally |
| `npm run lint` | Run ESLint |
| `npm run typecheck` | TypeScript type checking |

## Tech Stack

- [Vite 8](https://vite.dev/) + [React 19](https://react.dev/) + [TypeScript 5.9](https://www.typescriptlang.org/)
- [Tailwind CSS 4](https://tailwindcss.com/) + [Radix UI](https://www.radix-ui.com/)
- [Monaco Editor](https://microsoft.github.io/monaco-editor/) for YAML editing
- [XY Flow](https://xyflow.com/) for the agent canvas
- [Zustand](https://zustand.docs.pmnd.rs/) for state management

## Documentation

- [Hector Studio Guide](https://gohector.dev/guides/studio/): Features and usage
- [Hector Documentation](https://gohector.dev/): Full framework docs
- [API Reference](https://gohector.dev/reference/api/): A2A protocol endpoints

## License

[MIT](LICENSE)
