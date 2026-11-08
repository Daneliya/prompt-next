//go:build !windows

package desktop

import (
	"os/exec"
	"syscall"
)

func detachReplacement(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
