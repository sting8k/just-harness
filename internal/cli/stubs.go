package cli

import (
	"errors"
	"io"
)

// Temporary until S4/S5.
func cmdQuery(e *env, args []string) error { return errors.New("query: not implemented yet") }
func cmdCheck(e *env, args []string) error { return errors.New("check: not implemented yet") }
func cmdInstall(stdout io.Writer, args []string) error {
	return errors.New("install: not implemented yet")
}
