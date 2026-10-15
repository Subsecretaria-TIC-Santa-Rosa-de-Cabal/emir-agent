package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/api"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/auth"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/config"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/popup"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/storage"
)

// Version is injected via -ldflags at release time.
var Version = "0.1.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(Version)
		return
	}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <base64-announcement>\n", os.Args[0])
		os.Exit(1)
	}

	payload, err := base64.StdEncoding.DecodeString(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode announcement: %v\n", err)
		os.Exit(1)
	}

	var ann models.AgentAnnouncement
	if err := json.Unmarshal(payload, &ann); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal announcement: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	stateRepo := storage.NewStateRepository(cfg.StatePath)
	tokenStore := storage.NewSecureTokenStorage(cfg.StatePath)

	state, err := stateRepo.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load state: %v\n", err)
		os.Exit(1)
	}
	if state == nil || state.AgentToken == "" {
		fmt.Fprintln(os.Stderr, "agent not paired; cannot show announcement")
		os.Exit(1)
	}

	token, privateKey, err := tokenStore.Retrieve()
	if err != nil {
		fmt.Fprintf(os.Stderr, "retrieve credentials: %v\n", err)
		os.Exit(1)
	}
	if token == "" || privateKey == nil {
		fmt.Fprintln(os.Stderr, "credentials not found; please re-pair the agent")
		os.Exit(1)
	}

	keyPair, err := auth.KeyPairFromPrivateKey(privateKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load key pair: %v\n", err)
		os.Exit(1)
	}

	authBundle := auth.NewTokenAuth(token, keyPair)
	apiClient := api.NewClient(cfg.CoreURL, authBundle)

	if err := popup.Show(context.Background(), apiClient, ann); err != nil {
		fmt.Fprintf(os.Stderr, "show announcement: %v\n", err)
		os.Exit(1)
	}
}
