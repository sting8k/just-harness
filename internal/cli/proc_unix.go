//go:build !windows

package cli

import (
	"context"
	"os/exec"
	"syscall"
)

// shellCommand runs command via sh -c. With killTree (a --timeout is set) the
// shell leads its own process group and cancellation kills the whole group,
// so pipelines, `a && b` and background children die with it. Without a
// timeout the command stays in the terminal's group so Ctrl-C still reaches it.
func shellCommand(ctx context.Context, command string, killTree bool) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	if killTree {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	}
	return cmd
}
