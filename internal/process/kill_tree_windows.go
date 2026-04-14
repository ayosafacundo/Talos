//go:build windows

package process

import (
	"fmt"
	"os"
	"os/exec"
)

// killCmdTree terminates the process tree (Windows: taskkill /T). Falls back to Process.Kill.
func killCmdTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	c := exec.Command("taskkill", "/PID", fmt.Sprintf("%d", pid), "/T", "/F")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	_ = c.Run()
	_ = cmd.Process.Kill()
}
