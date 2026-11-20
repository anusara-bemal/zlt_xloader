// Package buildinfo holds build-time metadata that is injected with -ldflags
// by the Makefile and by GoReleaser. Keeping these variables in one place
// gives `zlt-unlocker version` a single, authoritative source of truth.
package buildinfo

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// AppName is the product name shown in help output and release artifacts.
const AppName = "zlt-unlocker"

var (
	// Version is the semantic version, e.g. v1.0.0. Defaults to "dev" for
	// local builds that do not inject a version.
	Version = "dev"

	// Commit is the full Git commit hash the binary was built from.
	Commit = "none"

	// Date is the UTC build timestamp in RFC 3339 format.
	Date = "unknown"
)

// String returns a human-readable, multi-line build report.
func String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", AppName, Version)
	fmt.Fprintf(&b, "  go:       %s\n", runtime.Version())
	fmt.Fprintf(&b, "  os/arch:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "  commit:   %s\n", Commit)
	fmt.Fprintf(&b, "  built:    %s\n", Date)
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Path != "" {
		fmt.Fprintf(&b, "  module:   %s\n", bi.Main.Path)
	}
	return b.String()
}
