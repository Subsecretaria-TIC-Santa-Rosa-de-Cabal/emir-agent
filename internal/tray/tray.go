// Package tray provides a system tray icon for the agent when running
// interactively. It is not used when the agent runs as a Windows service
// because services run in session 0 and cannot display UI.
package tray

import (
	"context"
	_ "embed"
	"fmt"

	"fyne.io/systray"
)

//go:embed favicon.ico
var faviconBytes []byte

// Run displays the system tray icon and blocks until the user selects Exit
// or the provided context is cancelled. The agentStop callback is invoked
// when the user requests exit so the agent lifecycle can shut down cleanly.
func Run(ctx context.Context, agentStop func()) error {
	var cancelAgent context.CancelFunc
	if agentStop == nil {
		cancelAgent = func() {}
	} else {
		cancelAgent = agentStop
	}

	ready := make(chan struct{})

	systray.Run(func() {
		defer close(ready)

		systray.SetIcon(faviconBytes)
		systray.SetTitle("EMIR Agent")
		systray.SetTooltip("EMIR Agent")

		mExit := systray.AddMenuItem("Cerrar agente", "Detener el agente EMIR")

		go func() {
			for {
				select {
				case <-mExit.ClickedCh:
					fmt.Println("tray: exit requested")
					cancelAgent()
					systray.Quit()
					return
				case <-ctx.Done():
					systray.Quit()
					return
				}
			}
		}()
	}, func() {
		// Cleanup on tray exit.
	})

	return ctx.Err()
}
