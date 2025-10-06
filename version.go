// Package hector provides version information for the Hector framework.
package hector

import (
	"fmt"
	"runtime"
)

// Version information
const (
	Version   = "0.1.0-alpha"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
	GitCommit string `json:"git_commit"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetVersion returns version information
func GetVersion() Info {
	return Info{
		Version:   Version,
		BuildDate: BuildDate,
		GitCommit: GitCommit,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf("Hector %s (built %s, commit %s, %s %s)",
		i.Version, i.BuildDate, i.GitCommit, i.GoVersion, i.Platform)
}
