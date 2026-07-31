// Package version holds release metadata injected at link time.
package version

import "fmt"

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func String() string {
	return fmt.Sprintf("spook %s (commit %s, built %s)", Version, Commit, BuildDate)
}
