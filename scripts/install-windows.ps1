#Requires -Version 5.1
<#
.SYNOPSIS
    Installs emir-agent on Windows as a service.
.DESCRIPTION
    Downloads the requested release, installs the binary under C:\Program Files\emir-agent,
    runs interactive pairing, registers a Windows service and starts it.
.PARAMETER CoreURL
    URL of the emir-core backend.
.PARAMETER Version
    Agent release version to install (e.g. 0.1.0).
.PARAMETER Repo
    GitHub repository owner/name where releases are published.
.PARAMETER InstallDir
    Directory where the agent binary will be installed.
#>
param(
    [string]$CoreURL = "",
    [string]$Version = "0.1.0",
    [string]$Repo = "alcaldia/emir-agent",
    [string]$InstallDir = "C:\Program Files\emir-agent"
)

$ErrorActionPreference = "Stop"

if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Error "This script must be run as Administrator."
    exit 1
}

$AssetName = "emir-agent-windows-amd64.zip"
$DownloadURL = "https://github.com/$Repo/releases/download/v$Version/$AssetName"
$ServiceName = "emir-agent"
$ServiceBinary = Join-Path $InstallDir "emir-agent.exe"

# Stop/remove any existing service so the binary can be replaced.
$ExistingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($ExistingService) {
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    sc.exe delete $ServiceName | Out-Null
    Start-Sleep -Seconds 2
}

$TempDir = Join-Path $env:TEMP "emir-agent-install-$Version"
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

Write-Host "Downloading $AssetName from GitHub..."
$ZipPath = Join-Path $TempDir $AssetName
Invoke-WebRequest -Uri $DownloadURL -OutFile $ZipPath -UseBasicParsing

Write-Host "Extracting to $InstallDir..."
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Expand-Archive -Path $ZipPath -DestinationPath $InstallDir -Force

$BinaryPath = Join-Path $InstallDir "emir-agent-windows-amd64.exe"
Move-Item -Path $BinaryPath -Destination $ServiceBinary -Force -ErrorAction SilentlyContinue

# Persist core URL as an environment variable for the service.
if ($CoreURL -ne "") {
    [Environment]::SetEnvironmentVariable("EMIR_CORE_URL", $CoreURL, "Machine")
    $env:EMIR_CORE_URL = $CoreURL
}

# Pair interactively before running as a service.
Write-Host "`nPairing the agent with emir-core..."
& $ServiceBinary --pair

if ($LASTEXITCODE -ne 0) {
    Write-Error "Pairing failed. The service will not be started."
    exit 1
}

Write-Host "Registering Windows service $ServiceName..."
sc.exe create $ServiceName binPath= "$ServiceBinary" start= auto DisplayName= "EMIR Agent" | Out-Null
sc.exe failure $ServiceName reset= 86400 actions= restart/60000/restart/60000/restart/60000 | Out-Null

Start-Service -Name $ServiceName
Write-Host "emir-agent installed and running."
