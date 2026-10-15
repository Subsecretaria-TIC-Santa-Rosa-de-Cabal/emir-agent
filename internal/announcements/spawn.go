package announcements

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/updater"
)

// popupBinaryName returns the popup executable name for the current platform.
func popupBinaryName() string {
	if runtime.GOOS == "windows" {
		return "emir-agent-popup.exe"
	}
	return "emir-agent-popup"
}

// SpawnAnnouncementViewer launches the emir-agent-popup binary as a detached
// process to display the given announcement. The popup binary must live in the
// same directory as the running agent executable.
func SpawnAnnouncementViewer(ann models.AgentAnnouncement) error {
	payload, err := json.Marshal(ann)
	if err != nil {
		return fmt.Errorf("marshal announcement: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(payload)

	agentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}

	popupPath := filepath.Join(filepath.Dir(agentPath), popupBinaryName())

	cmd := exec.Command(popupPath, encoded)
	return updater.StartDetached(cmd)
}
