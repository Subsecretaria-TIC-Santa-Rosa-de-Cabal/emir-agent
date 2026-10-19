#Requires -Version 5.1
<#
.SYNOPSIS
    Runs emir-agent in interactive mode for testing.
.DESCRIPTION
    Stops the Windows service (if it exists) and launches the agent binary
    interactively. In this mode the system tray icon is shown and logs are
    printed to the console, which is useful for debugging popups and polling.
.PARAMETER InstallDir
    Directory where emir-agent.exe is installed.
#>
param(
    [string]$InstallDir = "C:\Program Files\emir-agent"
)

$ErrorActionPreference = "Stop"

$ServiceName = "emir-agent"
$AgentBinary = Join-Path $InstallDir "emir-agent.exe"

if (-not (Test-Path $AgentBinary)) {
    Write-Error "Agent binary not found at $AgentBinary"
    exit 1
}

Write-Host "Stopping service $ServiceName (if running)..."
Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue

# Use the same machine-wide state path the service uses.
$Env:EMIR_STATE_PATH = Join-Path $env:PROGRAMDATA "emir-agent\state.json"

Write-Host "Starting emir-agent in interactive mode..."
Write-Host "Press Ctrl+C to stop."
& $AgentBinary
