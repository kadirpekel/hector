package ui

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var distFS embed.FS

// Handler returns an http.Handler that serves the UI.
// It serves from the embedded filesystem if real UI assets are present.
// If no real UI assets are embedded, a fallback page is shown.
func Handler() http.Handler {
	// Check if embedded FS has real UI assets (not just .gitkeep)
	if _, err := fs.Stat(distFS, "dist/index.html"); err == nil {
		sub, _ := fs.Sub(distFS, "dist")
		return spaFileServer(http.FS(sub))
	}

	return http.HandlerFunc(fallbackHandler)
}

// spaFileServer serves static files with SPA fallback to index.html.
func spaFileServer(root http.FileSystem) http.Handler {
	fileServer := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to open the requested path
		f, err := root.Open(r.URL.Path)
		if err != nil {
			// Not found - serve index.html for client-side routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	})
}

func fallbackHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, fallbackHTML)
}

const fallbackHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hector Studio</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 60px auto; padding: 0 20px; color: #333; }
        h1 { color: #111; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; font-size: 0.9em; }
        a { color: #0066cc; }
    </style>
</head>
<body>
    <h1>Hector Studio</h1>
    <p>The web UI is not included in this build.</p>
    <p>Install Hector with the Studio UI:</p>
    <ul>
        <li>macOS / Linux: <code>curl -fsSL https://gohector.dev/install.sh | sh</code></li>
        <li>Windows (PowerShell): <code>irm https://gohector.dev/install.ps1 | iex</code></li>
        <li>Homebrew: <code>brew install verikod/tap/hector</code></li>
    </ul>
    <p>Then run <code>hector serve</code> and open <b>http://localhost:8080</b>.</p>
    <p>Other options:</p>
    <ul>
        <li>Build from source with UI: <code>make build-release</code> (requires Node.js)</li>
        <li>Studio dev server: <code>cd studio &amp;&amp; npm run dev</code></li>
        <li>Try online: <a href="https://studio.gohector.dev">studio.gohector.dev</a></li>
    </ul>
    <p>The API server is running &mdash; agents are available at <code>/agents/</code>.</p>
    <p><a href="/health">/health</a> &middot; <a href="/agents">/agents</a></p>
</body>
</html>
`
