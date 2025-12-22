//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// killProcessGroup kills a process and all its children on Unix
func killProcessGroup(pid int) {
	syscall.Kill(-pid, syscall.SIGKILL)
}

// setupProcessGroup configures cmd to run in its own process group on Unix
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
