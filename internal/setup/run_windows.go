package setup

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// hideWindow keeps a console program from opening a console window.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
}
