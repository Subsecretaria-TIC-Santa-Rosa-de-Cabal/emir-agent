package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
	logPath := filepath.Join(os.TempDir(), "emir-agent-popup.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		defer logFile.Close()
		os.Stdout = logFile
		os.Stderr = logFile
	} else {
		fmt.Fprintf(os.Stderr, "failed to open popup log file %s: %v\n", logPath, err)
	}

	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(Version)
		return
	}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <base64-announcement>|--file <json-file>\n", os.Args[0])
		os.Exit(1)
	}

	var payload []byte
	if os.Args[1] == "--file" {
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "missing file path for --file")
			os.Exit(1)
		}
		payload, err = os.ReadFile(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "read announcement file: %v\n", err)
			os.Exit(1)
		}
	} else {
		payload, err = base64.StdEncoding.DecodeString(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "decode announcement: %v\n", err)
			os.Exit(1)
		}
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

	var apiClient *api.Client

	token, privateKey, err := tokenStore.Retrieve()
	if err != nil {
		fmt.Fprintf(os.Stderr, "retrieve credentials: %v\n", err)
	}
	if token == "" || privateKey == nil {
		fmt.Fprintln(os.Stderr, "credentials not found; popup will be shown without reporting impression")
	} else {
		keyPair, err := auth.KeyPairFromPrivateKey(privateKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load key pair: %v\n", err)
		} else {
			authBundle := auth.NewTokenAuth(token, keyPair)
			apiClient = api.NewClient(cfg.CoreURL, authBundle)
		}
	}

	fmt.Printf("popup starting for announcement %s\n", ann.Identifier)
	if err := popup.Show(context.Background(), apiClient, ann); err != nil {
		fmt.Fprintf(os.Stderr, "show announcement: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("popup closed")
}
