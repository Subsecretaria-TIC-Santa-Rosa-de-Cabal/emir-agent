package announcements

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
)

// popupBinaryName returns the popup executable name for the current platform.
func popupBinaryName() string {
	if runtime.GOOS == "windows" {
		return "emir-agent-popup.exe"
	}
	return "emir-agent-popup"
}

// SpawnAnnouncementViewer launches the emir-agent-popup binary to display the
// given announcement. On Windows the popup is started in the active user's
// session so it is visible when the agent is running as a service. On Unix it
// runs as a detached child process.
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

	if _, err := os.Stat(popupPath); err != nil {
		return fmt.Errorf("popup binary not found at %s: %w", popupPath, err)
	}

	logPath := filepath.Join(os.TempDir(), "emir-agent-popup.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open popup log: %w", err)
	}
	defer logFile.Close()

	fmt.Printf("spawning popup viewer: %s (log: %s)\n", popupPath, logPath)

	return startPopup(popupPath, encoded, logFile)
}
