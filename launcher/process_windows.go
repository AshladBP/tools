//go:build windows

package main

import (
	"fmt"
	"os/exec"
)

func setupProcessGroup(cmd *exec.Cmd) {
	// Windows handles process trees differently
	// taskkill /T will kill child processes
}

func killProcessGroup(pid int) {
	// Use taskkill with /T to kill process tree
	exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(pid)).Run()
}
