package agent

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/api"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/auth"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/collectors"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/config"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/pairing"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/storage"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/updater"
)

// Agent is the desktop agent runtime.
type Agent struct {
	config       *config.Config
	stateRepo    *storage.StateRepository
	tokenStore   *storage.SecureTokenStorage
	apiClient    *api.Client
	authBundle   *auth.TokenAuth
	keyPair      *auth.KeyPair
}

// New builds a new Agent.
func New(cfg *config.Config) (*Agent, error) {
	stateRepo := storage.NewStateRepository(cfg.StatePath)
	tokenStore := storage.NewSecureTokenStorage(cfg.StatePath)

	return &Agent{
		config:     cfg,
		stateRepo:  stateRepo,
		tokenStore: tokenStore,
	}, nil
}

// PairOnly runs the interactive pairing flow once and exits.
func (a *Agent) PairOnly(ctx context.Context) error {
	return a.pair(ctx)
}

// Run starts the agent lifecycle.
func (a *Agent) Run(ctx context.Context) error {
	fmt.Printf("emir-agent %s starting...\n", models.Version)
	fmt.Printf("state path: %s\n", a.config.StatePath)

	state, err := a.stateRepo.Load()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	if state == nil || state.AgentToken == "" {
		if err := a.pair(ctx); err != nil {
			return fmt.Errorf("pair: %w", err)
		}
	} else {
		// Defensive: if the state remembers a version newer than the binary's
		// compiled default, use it. This prevents update loops when a release
		// build is missing the -ldflags version override.
		if state.Version != "" && models.CompareVersions(state.Version, models.Version) > 0 {
			fmt.Printf("using version from state: %s (binary default: %s)\n", state.Version, models.Version)
			models.Version = state.Version
		}

		if err := a.loadAuth(state); err != nil {
			return fmt.Errorf("load auth: %w", err)
		}
	}

	a.apiClient = api.NewClient(a.config.CoreURL, a.authBundle)

	// Run first cycle immediately.
	if err := a.cycle(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "initial cycle failed: %v\n", err)
	}

	ticker := time.NewTicker(a.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := a.cycle(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "cycle failed: %v\n", err)
			}
		}
	}
}

func (a *Agent) pair(ctx context.Context) error {
	keyPair, err := auth.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate key pair: %w", err)
	}
	a.keyPair = keyPair

	coreURL, pairingCode, err := pairing.Prompt(a.config.CoreURL)
	if err != nil {
		return err
	}

	// Update core URL if technician changed it.
	a.config.CoreURL = coreURL

	// Pairing is unauthenticated; build a temporary client.
	tempClient := api.NewClient(coreURL, nil)
	resp, err := tempClient.Pair(ctx, pairing.BuildPairRequest(
		pairingCode,
		keyPair,
		ptrString(hostnameSafe()),
		nil,
		nil,
	))
	if err != nil {
		return fmt.Errorf("pair request: %w", err)
	}

	if err := a.tokenStore.Store(resp.AgentToken, a.keyPair.PrivateKeySeed()); err != nil {
		return fmt.Errorf("store credentials: %w", err)
	}

	state := &models.AgentState{
		ComputerIdentifier: resp.ComputerIdentifier,
		AgentToken:         resp.AgentToken,
		Version:            models.Version,
	}
	if err := a.stateRepo.Save(state); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	a.authBundle = auth.NewTokenAuth(resp.AgentToken, keyPair)
	return nil
}

func (a *Agent) loadAuth(state *models.AgentState) error {
	token, privateKey, err := a.tokenStore.Retrieve()
	if err != nil {
		return fmt.Errorf("retrieve credentials: %w", err)
	}
	if token == "" || privateKey == nil {
		// Credentials missing from secure store; require re-pairing.
		if err := a.stateRepo.Clear(a.tokenStore); err != nil {
			return fmt.Errorf("clear state: %w", err)
		}
		return fmt.Errorf("credentials not found in secure store; please re-pair")
	}

	keyPair, err := auth.KeyPairFromPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("load key pair: %w", err)
	}
	a.keyPair = keyPair
	a.authBundle = auth.NewTokenAuth(token, keyPair)

	// Update state file with current version.
	state.Version = models.Version
	if err := a.stateRepo.Save(state); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	return nil
}

func (a *Agent) cycle(ctx context.Context) error {
	if a.apiClient == nil {
		return fmt.Errorf("api client not initialized")
	}

	// Heartbeat first.
	heartbeatResp, err := a.apiClient.Heartbeat(ctx)
	if err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	fmt.Printf("heartbeat OK, next poll in %ds\n", heartbeatResp.NextPollInSeconds)

	// Inventory.
	inventory := collectors.Collect()
	if err := a.apiClient.Inventory(ctx, inventory); err != nil {
		return fmt.Errorf("inventory: %w", err)
	}
	fmt.Println("inventory synced")

	// Version check.
	versionResp, err := a.apiClient.Version(ctx)
	if err != nil {
		return fmt.Errorf("version check: %w", err)
	}

	downloadURL, checksum, hasAsset := versionResp.AssetForPlatform(runtime.GOOS)
	if versionResp.Version != models.Version {
		fmt.Printf("new agent version available: %s (current: %s)\n", versionResp.Version, models.Version)
		if !hasAsset {
			fmt.Printf("no asset available for platform %s, skipping auto-update\n", runtime.GOOS)
			return nil
		}
		if versionResp.IsMandatory {
			fmt.Printf("applying mandatory update from %s...\n", downloadURL)
			// Remember the target version before attempting the update. If the
			// new binary is built without the correct -ldflags version override,
			// this prevents an infinite update loop on restart.
			if state, err := a.stateRepo.Load(); err == nil && state != nil {
				state.Version = versionResp.Version
				if saveErr := a.stateRepo.Save(state); saveErr != nil {
					fmt.Fprintf(os.Stderr, "failed to save target version: %v\n", saveErr)
				}
			}
			if err := updater.Apply(*versionResp); err != nil {
				return fmt.Errorf("apply update: %w", err)
			}
			// The updater script will replace the binary and restart the service.
			return nil
		}
		fmt.Printf("optional update available (checksum: %s)\n", checksum)
	}

	return nil
}

func hostnameSafe() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func ptrString(s string) *string {
	return &s
}
