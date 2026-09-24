// Package cli parses arguments (stdlib flag) and dispatches commands.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sting8k/just-harness/internal/store"
)

// Version is stamped at build time via -ldflags -X; "dev" for local builds.
var Version = "dev"

// inputError marks bad usage: exit 2.
type inputError struct{ msg string }

func (e inputError) Error() string { return e.msg }

func inputf(format string, a ...any) error { return inputError{fmt.Sprintf(format, a...)} }

// exitCode propagates a child exit code (story verify) without a message.
type exitCode int

func (e exitCode) Error() string { return fmt.Sprintf("exit %d", int(e)) }

// Run dispatches one invocation and returns the process exit code:
// 0 ok, 2 input error, 1 runtime error / invariant refusal.
func Run(args []string, stdout, stderr io.Writer) int {
	err := dispatch(args, stdout, stderr)
	var ec exitCode
	var ie inputError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &ec):
		return int(ec)
	case errors.As(err, &ie):
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	default:
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
}

type handler func(env *env, args []string) error

// env carries the writers and (lazily) the store for one invocation.
type env struct {
	stdout, stderr io.Writer
	st             store.Store
}

var subcommands = map[string]map[string]handler{
	"story":     {"add": storyAdd, "update": storyUpdate, "verify": storyVerify},
	"decision":  {"add": decisionAdd, "update": decisionUpdate},
	"guardrail": {"add": guardrailAdd, "supersede": guardrailSupersede},
}

var commands = map[string]handler{
	"trace": cmdTrace,
	"query": cmdQuery,
	"check": cmdCheck,
}

func dispatch(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(stdout, helpTop)
		return nil
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "help", "-h", "--help":
		return printHelp(stdout, rest)
	case "version", "-v", "--version":
		fmt.Fprintln(stdout, Version)
		return nil
	case "install":
		return cmdInstall(stdout, rest)
	}
	e := &env{stdout: stdout, stderr: stderr}
	h, ok := commands[cmd]
	if subs, isGroup := subcommands[cmd]; isGroup {
		if len(rest) == 0 || rest[0] == "-h" || rest[0] == "--help" {
			return printHelp(stdout, []string{cmd})
		}
		if h, ok = subs[rest[0]]; !ok {
			return inputf("unknown %s action %q (see `help %s`)", cmd, rest[0], cmd)
		}
		rest = rest[1:]
	} else if !ok {
		return inputf("unknown command %q (see `help`)", cmd)
	}
	if hasHelpFlag(rest) {
		return printHelp(stdout, []string{cmd})
	}
	root, err := store.FindRoot(".")
	if err != nil {
		return err
	}
	e.st = store.Store{Root: root}
	return h(e, rest)
}

func hasHelpFlag(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "--help" || a == "-help" {
			return true
		}
	}
	return false
}

// flags wraps a FlagSet and remembers which flags were explicitly set.
type flags struct {
	fs  *flag.FlagSet
	set map[string]bool
}

func newFlags(name string) *flags {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return &flags{fs: fs}
}

func (f *flags) String(name string) *string { return f.fs.String(name, "", "") }
func (f *flags) Bool(name string) *bool     { return f.fs.Bool(name, false, "") }

// parse parses args, rejects positionals and checks required flags.
func (f *flags) parse(args []string, required ...string) error {
	if err := f.fs.Parse(args); err != nil {
		return inputf("%v", err)
	}
	if f.fs.NArg() > 0 {
		return inputf("unexpected argument %q", f.fs.Arg(0))
	}
	f.set = map[string]bool{}
	f.fs.Visit(func(fl *flag.Flag) { f.set[fl.Name] = true })
	for _, r := range required {
		if strings.TrimSpace(f.fs.Lookup(r).Value.String()) == "" {
			return inputf("--%s is required", r)
		}
	}
	return nil
}

// opt returns a pointer to the flag value when set, else nil.
func (f *flags) opt(name string) *string {
	if !f.set[name] {
		return nil
	}
	v := f.fs.Lookup(name).Value.String()
	return &v
}

// checkRepoPath refuses a repo-relative path that does not exist.
func (e *env) checkRepoPath(flagName, rel string) error {
	if rel == "" {
		return nil
	}
	if filepath.IsAbs(rel) {
		return inputf("--%s must be relative to the repo root: %s", flagName, rel)
	}
	if _, err := os.Stat(filepath.Join(e.st.Root, filepath.FromSlash(rel))); err != nil {
		return inputf("--%s: %s does not exist (paths are relative to the repo root)", flagName, rel)
	}
	return nil
}
