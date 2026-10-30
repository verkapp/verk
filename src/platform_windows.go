//go:build windows

package src

import (
	"os/exec"
	"syscall"
)

// hideWindow prevents console windows from flashing when Claude Code is
// spawned from the GUI application.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
