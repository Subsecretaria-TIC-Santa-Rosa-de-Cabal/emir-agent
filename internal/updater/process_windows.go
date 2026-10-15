//go:build windows

package updater

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// StartDetached launches a process independent of the parent so it survives
// service stop / parent exit on Windows.
func StartDetached(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
	}
	return cmd.Start()
}
