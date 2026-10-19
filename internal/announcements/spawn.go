package announcements

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

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

	// Write the announcement to a temporary file and pass the path to the popup.
	// This avoids command-line length limits and escaping issues with long URLs.
	announcementFile := filepath.Join(os.TempDir(), fmt.Sprintf("emir-announcement-%d.json", time.Now().UnixNano()))
	if err := os.WriteFile(announcementFile, payload, 0644); err != nil {
		return fmt.Errorf("write announcement file: %w", err)
	}

	agentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}

	popupPath := filepath.Join(filepath.Dir(agentPath), popupBinaryName())

	if _, err := os.Stat(popupPath); err != nil {
		return fmt.Errorf("popup binary not found at %s: %w", popupPath, err)
	}

	fmt.Printf("spawning popup viewer: %s (announcement: %s)\n", popupPath, announcementFile)

	return startPopup(popupPath, announcementFile)
}
