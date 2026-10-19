//go:build windows

package main

import "golang.org/x/sys/windows"

// hasConsole reports whether the popup is attached to a console so stdout/stderr
// can be used for debugging. When there is no console (e.g. spawned from a
// Windows service) we redirect output to a log file.
func hasConsole() bool {
	var mode uint32
	if windows.GetConsoleMode(windows.Stdout, &mode) == nil {
		return true
	}
	if windows.GetConsoleMode(windows.Stderr, &mode) == nil {
		return true
	}
	return false
}
