//go:build !windows

package announcements

import (
	"os"
	"os/exec"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/updater"
)

// startPopup launches the popup binary as a detached process on Unix-like
// platforms. The log file handle is not used because the detached process
// cannot inherit stdio reliably.
func startPopup(popupPath string, encoded string, logFile *os.File) error {
	cmd := exec.Command(popupPath, encoded)
	return updater.StartDetached(cmd)
}
