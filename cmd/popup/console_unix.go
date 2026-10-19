//go:build !windows

package main

import "golang.org/x/sys/unix"

// hasConsole reports whether the popup is attached to a terminal so stdout/stderr
// can be used for debugging. When there is no terminal (e.g. spawned from a
// background service) we redirect output to a log file.
func hasConsole() bool {
	return unix.Isatty(unix.Stdout) || unix.Isatty(unix.Stderr)
}
