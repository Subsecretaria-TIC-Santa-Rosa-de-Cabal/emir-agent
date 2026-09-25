package models

import "testing"

func TestAgentVersionResponseAssetForPlatformWindows(t *testing.T) {
	resp := AgentVersionResponse{
		Version:     "0.2.0",
		DownloadURL: "https://example.com/windows.exe",
		Checksum:    "sha256-windows",
	}

	url, checksum, ok := resp.AssetForPlatform("windows")
	if !ok {
		t.Fatal("expected ok for windows")
	}
	if url != resp.DownloadURL {
		t.Fatalf("expected %s, got %s", resp.DownloadURL, url)
	}
	if checksum != resp.Checksum {
		t.Fatalf("expected %s, got %s", resp.Checksum, checksum)
	}
}

func TestAgentVersionResponseAssetForPlatformLinux(t *testing.T) {
	linuxURL := "https://example.com/linux"
	linuxChecksum := "sha256-linux"
	resp := AgentVersionResponse{
		Version:          "0.2.0",
		DownloadURL:      "https://example.com/windows.exe",
		Checksum:         "sha256-windows",
		LinuxDownloadURL: &linuxURL,
		LinuxChecksum:    &linuxChecksum,
	}

	url, checksum, ok := resp.AssetForPlatform("linux")
	if !ok {
		t.Fatal("expected ok for linux")
	}
	if url != linuxURL {
		t.Fatalf("expected %s, got %s", linuxURL, url)
	}
	if checksum != linuxChecksum {
		t.Fatalf("expected %s, got %s", linuxChecksum, checksum)
	}
}

func TestAgentVersionResponseAssetForPlatformMacOS(t *testing.T) {
	macosURL := "https://example.com/macos"
	macosChecksum := "sha256-macos"
	resp := AgentVersionResponse{
		Version:          "0.2.0",
		DownloadURL:      "https://example.com/windows.exe",
		Checksum:         "sha256-windows",
		MacOSDownloadURL: &macosURL,
		MacOSChecksum:    &macosChecksum,
	}

	url, checksum, ok := resp.AssetForPlatform("darwin")
	if !ok {
		t.Fatal("expected ok for darwin")
	}
	if url != macosURL {
		t.Fatalf("expected %s, got %s", macosURL, url)
	}
	if checksum != macosChecksum {
		t.Fatalf("expected %s, got %s", macosChecksum, checksum)
	}
}

func TestAgentVersionResponseAssetForPlatformUnsupported(t *testing.T) {
	resp := AgentVersionResponse{
		Version:     "0.2.0",
		DownloadURL: "https://example.com/windows.exe",
		Checksum:    "sha256-windows",
	}

	_, _, ok := resp.AssetForPlatform("freebsd")
	if ok {
		t.Fatal("expected not ok for unsupported platform")
	}
}

func TestAgentVersionResponseAssetForPlatformMissingLinux(t *testing.T) {
	resp := AgentVersionResponse{
		Version:     "0.2.0",
		DownloadURL: "https://example.com/windows.exe",
		Checksum:    "sha256-windows",
	}

	_, _, ok := resp.AssetForPlatform("linux")
	if ok {
		t.Fatal("expected not ok when linux asset is missing")
	}
}
