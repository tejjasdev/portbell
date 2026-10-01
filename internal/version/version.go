// Package version holds build information set at link time.
package version

import (
	"runtime/debug"
	"strings"
)

// These values are overridden by the release build (see .goreleaser.yaml and
// the Makefile) using -ldflags "-X ...".
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func init() {
	// Built with `go install module@version`: no linker flags, but Go records
	// the module version in the binary.
	if Version != "dev" {
		return
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			Version = strings.TrimPrefix(v, "v")
		}
	}
}

// String returns a one-line description such as "0.1.0 (abc1234, 2026-10-01)".
func String() string {
	return Version + " (" + Commit + ", " + Date + ")"
}
