package version

import "strings"

// Version and Commit are set at link time (see Makefile and release workflow).
var (
	Version = "dev"
	Commit  = ""
)

// Short returns a compact label for the UI (git tag or git describe output).
func Short() string {
	v := strings.TrimSpace(Version)
	if v == "" || v == "dev" {
		if c := strings.TrimSpace(Commit); c != "" {
			return "dev·" + c
		}
		return "dev"
	}
	return v
}

// String is the full line printed by -version.
func String() string {
	line := Short()
	if c := strings.TrimSpace(Commit); c != "" && !strings.Contains(line, c) {
		return line + " (" + c + ")"
	}
	return line
}
