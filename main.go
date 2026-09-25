package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/agent"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/config"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/service"
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

	if pairOnly {
		if err := ag.PairOnly(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "pair error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("pairing completed")
		return
	}

	if err := service.Run(ag); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "agent error: %v\n", err)
		os.Exit(1)
	}
}
