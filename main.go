package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/agent"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/config"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/service"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(models.Version)
		return
	}

	pairOnly := len(os.Args) > 1 && os.Args[1] == "--pair"
	updateCheck := len(os.Args) > 1 && os.Args[1] == "--update-check"

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

	if updateCheck {
		updated, err := ag.RunUpdateCheck(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "update check failed: %v\n", err)
			os.Exit(1)
		}
		if updated {
			fmt.Println("update initiated; the service will restart")
		} else {
			fmt.Println("no mandatory update available")
		}
		return
	}

	if len(os.Args) > 2 && os.Args[1] == "--show-announcement" {
		encoded := os.Args[2]
		payload, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			fmt.Fprintf(os.Stderr, "decode announcement: %v\n", err)
			os.Exit(1)
		}
		var ann models.AgentAnnouncement
		if err := json.Unmarshal(payload, &ann); err != nil {
			fmt.Fprintf(os.Stderr, "unmarshal announcement: %v\n", err)
			os.Exit(1)
		}
		if err := ag.ShowAnnouncement(context.Background(), ann); err != nil {
			fmt.Fprintf(os.Stderr, "show announcement failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := service.Run(ag); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "agent error: %v\n", err)
		os.Exit(1)
	}
}
