#Requires -Version 5.1
<#
.SYNOPSIS
    Build script for emir-agent on Windows.
.DESCRIPTION
    Cross-compiles emir-agent and emir-agent-popup for Windows amd64 and packages
    both binaries into a zip.
#>
param(
    [string]$Version = "0.1.0",
    [string]$OutputDir = "dist"
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$OutputDir = Join-Path $ProjectRoot $OutputDir

if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

$Env:GOOS = "windows"
$Env:GOARCH = "amd64"
$Env:GOTOOLCHAIN = "local"

$AgentBinaryName = "emir-agent-windows-amd64.exe"
$AgentBinaryPath = Join-Path $OutputDir $AgentBinaryName
$PopupBinaryName = "emir-agent-popup-windows-amd64.exe"
$PopupBinaryPath = Join-Path $OutputDir $PopupBinaryName

$Ldflags = "-s -w -X github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models.Version=$Version"
$PopupLdflags = "-s -w -X main.Version=$Version"

Write-Host "Building $AgentBinaryName (version $Version)..."
$Env:CGO_ENABLED = "0"
go build -ldflags "$Ldflags" -o $AgentBinaryPath $ProjectRoot
if (-not $?) {
    throw "Agent build failed"
}

Write-Host "Building $PopupBinaryName (version $Version)..."
go build -ldflags "$PopupLdflags" -o $PopupBinaryPath (Join-Path $ProjectRoot "cmd\popup")
if (-not $?) {
    throw "Popup build failed"
}

$BootScript = Join-Path $PSScriptRoot "update-at-boot.ps1"
$BootScriptDest = Join-Path $OutputDir "update-at-boot.ps1"
Copy-Item -Path $BootScript -Destination $BootScriptDest -Force

$AssetsSource = Join-Path $ProjectRoot "assets"
$AssetsDest = Join-Path $OutputDir "assets"
if (Test-Path $AssetsSource) {
    Copy-Item -Path $AssetsSource -Destination $AssetsDest -Recurse -Force
}

$ZipPath = Join-Path $OutputDir "emir-agent-windows-amd64.zip"
Compress-Archive -Path $AgentBinaryPath, $PopupBinaryPath, $BootScriptDest, $AssetsDest -DestinationPath $ZipPath -Force

Write-Host "Built: $AgentBinaryPath"
Write-Host "Built: $PopupBinaryPath"
Write-Host "Packaged: $ZipPath"
Write-Host "Included: $BootScriptDest"
Write-Host "Included: $AssetsDest"

# Compute checksums
$AgentHash = (Get-FileHash -Path $AgentBinaryPath -Algorithm SHA256).Hash.ToLower()
$AgentHash | Set-Content -Path "$AgentBinaryPath.sha256" -NoNewline
Write-Host "Agent checksum: $AgentHash"

$PopupHash = (Get-FileHash -Path $PopupBinaryPath -Algorithm SHA256).Hash.ToLower()
$PopupHash | Set-Content -Path "$PopupBinaryPath.sha256" -NoNewline
Write-Host "Popup checksum: $PopupHash"
