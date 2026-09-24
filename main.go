package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/emir/emir-agent/internal/agent"
	"github.com/emir/emir-agent/internal/config"
)

func main() {
	pairOnly := len(os.Args) > 1 && os.Args[1] == "--pair"

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	ag, err := agent.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create agent: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("shutting down...")
		cancel()
	}()

	if pairOnly {
		if err := ag.PairOnly(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "pair error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("pairing completed")
		return
	}

	if err := ag.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "agent error: %v\n", err)
		os.Exit(1)
	}
}
