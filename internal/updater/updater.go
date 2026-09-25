package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
)

// Apply downloads the new binary for the current platform, validates checksum, and triggers replacement.
func Apply(versionResp models.AgentVersionResponse) error {
	currentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get current executable: %w", err)
	}
	currentPath, err = filepath.EvalSymlinks(currentPath)
	if err != nil {
		return fmt.Errorf("eval symlinks: %w", err)
	}

	downloadURL, checksum, ok := versionResp.AssetForPlatform(runtime.GOOS)
	if !ok {
		return fmt.Errorf("no asset available for platform %s", runtime.GOOS)
	}

	tempDir, err := os.MkdirTemp("", "emir-agent-update-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}

	newBinaryPath := filepath.Join(tempDir, filepath.Base(currentPath)+".new")
	if err := download(downloadURL, newBinaryPath); err != nil {
		return fmt.Errorf("download update: %w", err)
	}

	if err := verifyChecksum(newBinaryPath, checksum); err != nil {
		return fmt.Errorf("checksum mismatch: %w", err)
	}

	if err := spawnUpdater(currentPath, newBinaryPath); err != nil {
		return fmt.Errorf("spawn updater: %w", err)
	}

	return nil
}

func download(url, dest string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func verifyChecksum(path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("expected %s, got %s", expected, actual)
	}
	return nil
}

func spawnUpdater(currentPath, newBinaryPath string) error {
	// Wait a moment for the parent process to exit, then replace and restart.
	switch runtime.GOOS {
	case "windows":
		return spawnWindowsUpdater(currentPath, newBinaryPath)
	case "darwin":
		return spawnDarwinUpdater(currentPath, newBinaryPath)
	default: // linux and others
		return spawnLinuxUpdater(currentPath, newBinaryPath)
	}
}

func spawnWindowsUpdater(currentPath, newBinaryPath string) error {
	script := fmt.Sprintf(`
$ErrorActionPreference = "Stop"
Start-Sleep -Seconds 2
$service = Get-Service -Name "emir-agent" -ErrorAction SilentlyContinue
if ($service) { Stop-Service -Name "emir-agent" -Force }
Move-Item -Path "%s" -Destination "%s" -Force
if ($service) { Start-Service -Name "emir-agent" } else { & "%s" }
Remove-Item -Path "%s" -Recurse -Force
`, newBinaryPath, currentPath, currentPath, filepath.Dir(newBinaryPath))

	scriptPath := filepath.Join(filepath.Dir(newBinaryPath), "update.ps1")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return err
	}

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", scriptPath)
	return cmd.Start()
}

func spawnLinuxUpdater(currentPath, newBinaryPath string) error {
	script := fmt.Sprintf(`#!/bin/sh
sleep 2
systemctl stop emir-agent || true
mv -f "%s" "%s"
chmod +x "%s"
systemctl start emir-agent
rm -rf "%s"
`, newBinaryPath, currentPath, currentPath, filepath.Dir(newBinaryPath))

	scriptPath := filepath.Join(filepath.Dir(newBinaryPath), "update.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return err
	}

	cmd := exec.Command("sh", scriptPath)
	return cmd.Start()
}

func spawnDarwinUpdater(currentPath, newBinaryPath string) error {
	script := fmt.Sprintf(`#!/bin/sh
sleep 2
launchctl unload /Library/LaunchDaemons/com.emir.agent.plist || true
mv -f "%s" "%s"
chmod +x "%s"
launchctl load /Library/LaunchDaemons/com.emir.agent.plist
rm -rf "%s"
`, newBinaryPath, currentPath, currentPath, filepath.Dir(newBinaryPath))

	scriptPath := filepath.Join(filepath.Dir(newBinaryPath), "update.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return err
	}

	cmd := exec.Command("sh", scriptPath)
	return cmd.Start()
}
