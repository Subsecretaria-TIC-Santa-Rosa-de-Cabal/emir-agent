//go:build !windows

package main

// hasConsole always reports false on Unix-like platforms. The popup is always
// launched as a detached process by the agent, so redirecting output to a log
// file is the safest default. This avoids importing platform-specific terminal
// detection packages.
func hasConsole() bool {
	return false
}
