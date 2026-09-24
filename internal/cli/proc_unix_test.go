//go:build !windows

package cli

import (
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStoryVerifyTimeoutKillsBackgroundChild(t *testing.T) {
	root := newRepo(t)
	id := mustRun(t, "story", "add", "--title", "bg", "--lane", "tiny", "--no-packet",
		"--verify", "sleep 30 & echo $! > child.pid; wait")
	if code, _, _ := run(t, "story", "verify", "--id", id, "--timeout", "300ms"); code != timeoutExitCode {
		t.Fatalf("timeout exit %d", code)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(readFile(t, root, "child.pid")))
	if err != nil {
		t.Fatal(err)
	}
	// The orphan is reaped by init shortly after the kill; poll briefly.
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if syscall.Kill(pid, 0) == syscall.ESRCH {
			return
		}
	}
	syscall.Kill(pid, syscall.SIGKILL)
	t.Fatalf("background child %d survived the timeout", pid)
}
