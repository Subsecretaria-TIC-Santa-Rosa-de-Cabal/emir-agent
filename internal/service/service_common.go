//go:build windows

package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/agent"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/tray"
)

func runInteractive(ag *agent.Agent) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
			fmt.Println("shutting down...")
			cancel()
		case <-ctx.Done():
		}
	}()

	// Run the agent lifecycle in the background and let the system tray icon
	// own the main thread. The tray provides an "Exit" option that cancels ctx.
	go func() {
		if err := ag.Run(ctx); err != nil && err != context.Canceled {
			fmt.Fprintf(os.Stderr, "agent error: %v\n", err)
			cancel()
		}
	}()

	return tray.Run(ctx, cancel)
}
