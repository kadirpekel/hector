// Package hector provides build-time version information.
// These variables are populated via ldflags during build.
package hector

// Version information set at build time via ldflags.
// When building with make: make build (uses git describe --tags)
// When running unbuilt (e.g. go run): falls back to the constant below.
var (
	// Version is the semantic version (e.g., "v1.20.0").
	// Set via: -X 'github.com/verikod/hector.Version=$(VERSION)'
	// VERSION in the Makefile is derived from: git describe --tags --always --dirty
	Version = "v1.20.0"

	// GitCommit is the short git commit hash.
	// Set via: -X 'github.com/verikod/hector.GitCommit=$(GIT_COMMIT)'
	GitCommit = "unknown"

	// BuildDate is the build timestamp in ISO 8601 format.
	// Set via: -X 'github.com/verikod/hector.BuildDate=$(BUILD_DATE)'
	BuildDate = "unknown"
)
