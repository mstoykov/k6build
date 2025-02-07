// Package version provides the version of the build
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// BuildVersion is the version of the build.
var BuildVersion = "dev" //nolint:gochecknoglobals

// Details returns the details about version and build information.
func Details() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	var (
		commit string
		dirty  bool
	)
	for _, s := range buildInfo.Settings {
		switch s.Key {
		case "vcs.revision":
			commitLen := 10
			if len(s.Value) < commitLen {
				commitLen = len(s.Value)
			}
			commit = s.Value[:commitLen]
		case "vcs.modified":
			if s.Value == "true" {
				dirty = true
			}
		default:
		}
	}

	if commit != "" {
		commit = fmt.Sprintf("commit/%s", commit)
		if dirty {
			commit += "-dirty"
		}
	}

	goVersionArch := fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	return fmt.Sprintf("%s %s %s", BuildVersion, commit, goVersionArch)
}
