//go:build windows

package cli

import (
	"context"
	"os/exec"
	"strconv"
)

// shellCommand runs command via cmd /C. Cancellation (--timeout) kills the
// whole process tree with taskkill /T, not just cmd.exe, so children do not
// outlive the verify run or keep its working directory busy.
func shellCommand(ctx context.Context, command string, killTree bool) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "cmd", "/C", command)
	if killTree {
		cmd.Cancel = func() error {
			if err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run(); err != nil {
				return cmd.Process.Kill()
			}
			return nil
		}
	}
	return cmd
}
