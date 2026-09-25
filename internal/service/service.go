// Package service provides OS-native service execution for the agent.
//
// On Windows it integrates with the Service Control Manager (SCM) so the
// agent can be registered and started with sc.exe. On Unix-like systems it
// simply runs the agent in the foreground and handles signals.
package service

import "github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/agent"

// Runner is implemented by platform-specific service runners.
type Runner interface {
	Run(ag *agent.Agent) error
}
