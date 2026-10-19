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

// preferredAnnouncementTempDir returns the ideal shared directory for
// announcement JSON files. On Windows the popup runs as the logged-on user
// while the service runs as SYSTEM, so ProgramData is preferred.
func preferredAnnouncementTempDir() string {
	switch runtime.GOOS {
	case "windows":
		return `C:\ProgramData\emir-agent\announcements`
	case "darwin":
		return "/Library/Application Support/emir-agent/announcements"
	default:
		return "/var/lib/emir-agent/announcements"
	}
}

// announcementTempDir returns a directory the current process can write to and
// the popup process can read from. It prefers the machine-wide location, but
// falls back to the user's temp directory when running interactively without
// elevation (e.g. local testing).
func announcementTempDir() string {
	preferred := preferredAnnouncementTempDir()
	if err := os.MkdirAll(preferred, 0755); err == nil {
		testFile := filepath.Join(preferred, ".write-test")
		if f, err := os.Create(testFile); err == nil {
			f.Close()
			_ = os.Remove(testFile)
			return preferred
		}
	}
	return os.TempDir()
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

	// Write the announcement to a shared file and pass the path to the popup.
	// This avoids command-line length limits and escaping issues with long URLs,
	// and uses a directory readable by the popup process on every platform.
	tempDir := announcementTempDir()
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("create announcement temp dir: %w", err)
	}
	announcementFile := filepath.Join(tempDir, fmt.Sprintf("emir-announcement-%d.json", time.Now().UnixNano()))
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
