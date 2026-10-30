//go:build !windows

package src

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
