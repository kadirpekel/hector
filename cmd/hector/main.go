// Command hector is the CLI for Hector.
//
// Clean architecture with strict separation:
//   - Server config (operational): CLI flags + env vars ONLY
//   - App config (functional): YAML files ONLY
//
// Usage:
//
//	hector serve                         # Start server (auto-creates config if needed)
//	hector init                          # Create app config
//	hector validate --config app.yaml    # Validate config
//	hector version                       # Show version
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/alecthomas/kong"
	hector "github.com/verikod/hector"
)

// CLI defines the root command structure.
// Simple: CLI is for default app only. Multi-app = Admin API.
type CLI struct {
	Version  VersionCmd  `cmd:"" help:"Show version information"`
	Serve    ServeCmd    `cmd:"" help:"Start the Hector server"`
	Init     InitCmd     `cmd:"" help:"Create a new app config file"`
	Validate ValidateCmd `cmd:"" help:"Validate an app config file"`
}

// VersionCmd shows version information.
type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	version := hector.Version

	// Fallback to module version (for go install)
	if version == "dev" || version == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "(devel)" && info.Main.Version != "" {
				version = info.Main.Version
			}
		}
	}

	fmt.Printf("Hector %s", version)
	if hector.GitCommit != "unknown" && hector.GitCommit != "" {
		fmt.Printf(" (%s)", hector.GitCommit)
	}
	if hector.BuildDate != "unknown" && hector.BuildDate != "" {
		fmt.Printf(" built %s", hector.BuildDate)
	}
	fmt.Println()
	return nil
}

func printBanner() {
	// Check if stdout is a terminal
	if fileInfo, err := os.Stdout.Stat(); err == nil {
		if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
			return // Not a terminal, skip banner
		}
	} else {
		return
	}

	greenColor := "\033[38;2;16;185;129m"
	resetColor := "\033[0m"

	banner := `
██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗
██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
`
	fmt.Printf("%s%s%s\n", greenColor, banner, resetColor)
}

func main() {
	// Skip banner for informational commands
	skipBanner := false
	for _, arg := range os.Args {
		if arg == "version" || arg == "init" || arg == "validate" {
			skipBanner = true
			break
		}
	}

	if !skipBanner {
		printBanner()
	}

	cli := CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("hector"),
		kong.Description("Hector - Config-first AI Agent Platform"),
		kong.UsageOnError(),
	)

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
