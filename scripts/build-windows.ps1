#Requires -Version 5.1
<#
.SYNOPSIS
    Build script for emir-agent on Windows.
.DESCRIPTION
    Cross-compiles emir-agent for Windows amd64 and packages it into a zip.
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
$Env:CGO_ENABLED = "0"

$BinaryName = "emir-agent-windows-amd64.exe"
$BinaryPath = Join-Path $OutputDir $BinaryName

Write-Host "Building $BinaryName (version $Version)..."
go build -ldflags "-s -w -X github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models.Version=$Version" -o $BinaryPath $ProjectRoot

if (-not $?) {
    throw "Build failed"
}

$ZipPath = Join-Path $OutputDir "emir-agent-windows-amd64.zip"
Compress-Archive -Path $BinaryPath -DestinationPath $ZipPath -Force

Write-Host "Built: $BinaryPath"
Write-Host "Packaged: $ZipPath"

# Compute checksum
$Hash = (Get-FileHash -Path $BinaryPath -Algorithm SHA256).Hash.ToLower()
$Hash | Set-Content -Path "$BinaryPath.sha256" -NoNewline
Write-Host "Checksum: $Hash"
