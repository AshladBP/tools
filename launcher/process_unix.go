//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessGroup(pid int) {
	// First try graceful shutdown
	syscall.Kill(-pid, syscall.SIGTERM)

	// Note: In production, you might want to wait and then SIGKILL
	// For now, we'll let the process handle SIGTERM
}
