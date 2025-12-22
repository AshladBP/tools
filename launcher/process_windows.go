//go:build windows

package main

import (
	"fmt"
	"os/exec"
)

// killProcessGroup kills a process and all its children on Windows
func killProcessGroup(pid int) {
	exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(pid)).Run()
}

// setupProcessGroup is a no-op on Windows (process groups work differently)
func setupProcessGroup(cmd *exec.Cmd) {
	// Windows doesn't use Setpgid, process tree killing is handled by taskkill /T
}
