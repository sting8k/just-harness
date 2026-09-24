package cli

import (
	"io"

	"github.com/sting8k/just-harness/internal/install"
)

// cmdInstall: install [path] [flags]; flags may come before or after path.
func cmdInstall(stdout io.Writer, args []string) error {
	f := newFlags("install")
	var o install.Options
	f.fs.BoolVar(&o.Merge, "merge", false, "")
	f.fs.BoolVar(&o.Override, "override", false, "")
	f.fs.BoolVar(&o.Force, "force", false, "")
	f.fs.BoolVar(&o.DryRun, "dry-run", false, "")
	f.fs.BoolVar(&o.Claude, "claude", false, "")
	var positional []string
	for {
		if err := f.fs.Parse(args); err != nil {
			return inputf("%v", err)
		}
		if f.fs.NArg() == 0 {
			break
		}
		positional = append(positional, f.fs.Arg(0))
		args = f.fs.Args()[1:]
	}
	if len(positional) > 1 {
		return inputf("expected at most one target directory, got %d", len(positional))
	}
	if o.Merge && o.Override {
		return inputf("--merge and --override are mutually exclusive")
	}
	o.Target = "."
	if len(positional) == 1 {
		o.Target = positional[0]
	}
	plan, err := install.NewPlan(o)
	if err != nil {
		return err
	}
	return plan.Execute(stdout)
}
