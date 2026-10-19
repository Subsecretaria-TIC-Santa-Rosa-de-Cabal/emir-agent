//go:build !windows

package announcements

import (
	"os"
	"os/exec"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/updater"
)

// startPopup launches the popup binary as a detached process on Unix-like
// platforms.
func startPopup(popupPath string, announcementFile string) error {
	cmd := exec.Command(popupPath, "--file", announcementFile)
	return updater.StartDetached(cmd)
}
