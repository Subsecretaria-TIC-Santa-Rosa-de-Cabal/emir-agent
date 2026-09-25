package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
)

func TestVerifyChecksumValid(t *testing.T) {
	content := []byte("hello world")
	expectedHash := sha256.Sum256(content)
	expectedChecksum := hex.EncodeToString(expectedHash[:])

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.bin")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if err := verifyChecksum(path, expectedChecksum); err != nil {
		t.Fatalf("expected valid checksum, got error: %v", err)
	}
}

func TestVerifyChecksumInvalid(t *testing.T) {
	content := []byte("hello world")

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.bin")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if err := verifyChecksum(path, "invalidchecksum"); err == nil {
		t.Fatal("expected error for invalid checksum")
	}
}

func TestApplyMissingAsset(t *testing.T) {
	resp := models.AgentVersionResponse{
		Version:     "0.2.0",
		DownloadURL: "",
		Checksum:    "",
	}

	err := Apply(resp)
	if err == nil {
		t.Fatal("expected error when no asset is available for platform")
	}
}
