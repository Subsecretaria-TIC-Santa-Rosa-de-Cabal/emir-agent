//go:build !windows

package updater

import "os/exec"

// startDetached launches cmd on Unix-like systems. Process groups are handled
// by the shell script itself so no special attributes are required.
func startDetached(cmd *exec.Cmd) error {
	return cmd.Start()
}
