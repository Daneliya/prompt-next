package desktop

import (
	"os/exec"
	"syscall"
)

func detachReplacement(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP, HideWindow: true}
}
