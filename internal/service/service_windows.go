//go:build windows

package service

import (
	"context"
	"fmt"
	"os"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/agent"
	"golang.org/x/sys/windows/svc"
)

type windowsService struct {
	agent  *agent.Agent
	cancel context.CancelFunc
}

func (s *windowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	defer cancel()

	go func() {
		if err := s.agent.Run(ctx); err != nil && err != context.Canceled {
			fmt.Fprintf(os.Stderr, "agent error: %v\n", err)
			os.Exit(1)
		}
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				return false, 0
			default:
			}
		case <-ctx.Done():
			return false, 0
		}
	}
}

// Run starts the agent. If the process is running interactively it behaves
// like a normal console application. Otherwise it registers with the Windows
// Service Control Manager.
func Run(ag *agent.Agent) error {
	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		return fmt.Errorf("determine interactive session: %w", err)
	}

	if isInteractive {
		return runInteractive(ag)
	}

	return svc.Run("emir-agent", &windowsService{agent: ag})
}
