//go:build !windows

package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/agent"
)

// Run starts the agent in the foreground and handles OS signals.
func Run(ag *agent.Agent) error {
	return runInteractive(ag)
}

func runInteractive(ag *agent.Agent) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("shutting down...")
		cancel()
	}()

	return ag.Run(ctx)
}
