//go:build !windows

package updater

import "os/exec"

// StartDetached launches cmd on Unix-like systems. Process groups are handled
// by the shell script itself so no special attributes are required.
func StartDetached(cmd *exec.Cmd) error {
	return cmd.Start()
}
