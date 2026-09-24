// Package cli parses arguments (stdlib flag) and dispatches commands.
package cli

import (
	"fmt"
	"io"
)

// Version is stamped at build time via -ldflags -X; "dev" for local builds.
var Version = "dev"

// Run dispatches one invocation and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stdout, "just-harness-cli")
		return 0
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, Version)
		return 0
	}
	fmt.Fprintf(stderr, "error: unknown command %q\n", args[0])
	return 2
}
