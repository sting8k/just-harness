// Command just-harness-cli is the single-binary just-harness engine.
package main

import (
	"os"

	"github.com/sting8k/just-harness/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
