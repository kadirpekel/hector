package cli

import (
	"fmt"
	"os"
)

// Fatalf prints an error message and exits with status 1
func Fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
