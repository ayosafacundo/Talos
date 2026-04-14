//go:build !windows

package process

import (
	"os/exec"
	"syscall"
	"time"
)

// killCmdTree terminates the process and its children. The command must have been
// started with Setpgid so the child's PID is the process group ID.
func killCmdTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		_ = cmd.Process.Kill()
		return
	}
	time.Sleep(500 * time.Millisecond)
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = cmd.Process.Kill()
}
