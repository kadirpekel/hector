package cli

import (
	"fmt"
	"os"
	"strings"
)

// CheckForMisplacedFlags detects flags after positional arguments and provides helpful error messages
func CheckForMisplacedFlags(args []string, command string) {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			Fatalf(`❌ Error: Flag '%s' appears after positional arguments

Flags must come BEFORE positional arguments in Go flag parsing.

WRONG:  hector %s agent %s
RIGHT:  hector %s %s agent

Common flags:
  --provider openai|anthropic|gemini
  --api-key KEY
  --model MODEL
  --base-url URL
  --tools
  --mcp-url URL

Run 'hector %s --help' for full usage.`, arg, command, arg, command, arg, command)
		}
	}
}

// Fatalf prints an error message and exits with status 1
func Fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
