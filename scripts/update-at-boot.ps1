#Requires -RunAsAdministrator
<#
.SYNOPSIS
    EMIR Agent update check at boot (Windows).
.DESCRIPTION
    Runs when the computer starts. If the agent binary is present it invokes
    emir-agent.exe --update-check to apply mandatory updates. If the binary is
    missing it enters recovery mode and re-downloads the version recorded at
    install time.
.PARAMETER InstallDir
    Directory where emir-agent.exe is installed.
.PARAMETER LogDir
    Directory where update-boot.log is written.
.PARAMETER LockTimeoutMinutes
    How old the lock file must be before this script ignores it.
#>
param(
    [string]$InstallDir = "C:\Program Files\emir-agent",
    [string]$LogDir = "$env:PROGRAMDATA\emir-agent",
    [int]$LockTimeoutMinutes = 5
)

$ErrorActionPreference = "Stop"

# GitHub requires TLS 1.2; PowerShell 5.1 defaults to older protocols on some systems.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$Binary = Join-Path $InstallDir "emir-agent.exe"
$LogFile = Join-Path $LogDir "update-boot.log"
$LockFile = Join-Path $LogDir "update.lock"
$MetaFile = Join-Path $InstallDir "install.conf"

function Write-Log {
    param([string]$Message)
    $line = "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') $Message"
    Add-Content -Path $LogFile -Value $line -ErrorAction SilentlyContinue
}

function Read-InstallMeta {
    $meta = @{
        Repo = "Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent"
        Version = ""
        Archive = "emir-agent-windows-amd64.zip"
    }
    if (Test-Path $MetaFile) {
        Get-Content $MetaFile | ForEach-Object {
            if ($_ -match '^\s*(\w+)\s*=\s*(.+?)\s*$') {
                $meta[$matches[1]] = $matches[2]
            }
        }
    }
    return $meta
}

function Download-File {
    param(
        [string]$Url,
        [string]$OutFile
    )
    $maxAttempts = 3
    for ($attempt = 1; $attempt -le $maxAttempts; $attempt++) {
        try {
            Invoke-WebRequest -Uri $Url -OutFile $OutFile -UseBasicParsing -Headers @{
                'User-Agent' = 'emir-agent-boot-updater'
            } -ErrorAction Stop
            return
        } catch {
            Write-Log "Download attempt $attempt of $maxAttempts failed: $_"
            if ($attempt -eq $maxAttempts) {
                $curl = Get-Command curl.exe -ErrorAction SilentlyContinue
                if (-not $curl) {
                    throw "Neither Invoke-WebRequest nor curl.exe is available"
                }
                & curl.exe -fsSL -A 'emir-agent-boot-updater' -o $OutFile $Url
                if ($LASTEXITCODE -ne 0) {
                    throw "curl.exe failed to download $Url"
                }
                return
            }
            Start-Sleep -Seconds 2
        }
    }
}

function Recover-Agent {
    $meta = Read-InstallMeta
    if (-not $meta.Version) {
        throw "Cannot recover agent: no version recorded in $MetaFile"
    }

    $tagVersion = $meta.Version
    if (-not $tagVersion.StartsWith('v')) {
        $tagVersion = "v$tagVersion"
    }

    $downloadUrl = "https://github.com/$($meta.Repo)/releases/download/$tagVersion/$($meta.Archive)"
    $checksumUrl = "$downloadUrl.sha256"

    Write-Log "Recovery mode: downloading $($meta.Version) from $downloadUrl"

    $tempDir = Join-Path $env:TEMP "emir-agent-recovery-$([Guid]::NewGuid())"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    try {
        $tempArchive = Join-Path $tempDir $meta.Archive
        Download-File -Url $downloadUrl -OutFile $tempArchive

        try {
            $expectedHash = (Invoke-WebRequest -Uri $checksumUrl -UseBasicParsing -Headers @{
                'User-Agent' = 'emir-agent-boot-updater'
            } -ErrorAction Stop).Content.Trim()
            $actualHash = (Get-FileHash -Path $tempArchive -Algorithm SHA256).Hash.ToLower()
            if ($actualHash -ne $expectedHash.ToLower()) {
                throw "Checksum mismatch: expected $expectedHash, got $actualHash"
            }
            Write-Log "Recovery checksum verified"
        } catch {
            Write-Log "Could not verify recovery checksum: $_"
        }

        Expand-Archive -Path $tempArchive -DestinationPath $tempDir -Force
        $agentSource = Join-Path $tempDir "emir-agent-windows-amd64.exe"
        $popupSource = Join-Path $tempDir "emir-agent-popup-windows-amd64.exe"
        $assetsSource = Join-Path $tempDir "assets"

        if (-not (Test-Path $agentSource)) {
            throw "Agent binary not found in recovered archive"
        }

        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        Move-Item -Path $agentSource -Destination $Binary -Force
        if (Test-Path $popupSource) {
            Move-Item -Path $popupSource -Destination (Join-Path $InstallDir "emir-agent-popup.exe") -Force
        }
        if (Test-Path $assetsSource) {
            $assetsDest = Join-Path $InstallDir "assets"
            New-Item -ItemType Directory -Path $assetsDest -Force | Out-Null
            Get-ChildItem -Path $assetsSource -Recurse | ForEach-Object {
                $dest = $_.FullName.Replace($assetsSource, $assetsDest)
                if ($_.PSIsContainer) {
                    New-Item -ItemType Directory -Path $dest -Force | Out-Null
                } else {
                    Copy-Item -Path $_.FullName -Destination $dest -Force
                }
            }
        }
        Write-Log "Agent restored to $InstallDir"
    } finally {
        Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Ensure directories exist.
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
New-Item -ItemType Directory -Path $LogDir -Force | Out-Null

Write-Log "Starting EMIR Agent boot update check"

# Lock file to avoid clashing with a running service update.
if (Test-Path $LockFile) {
    $lockAge = (Get-Date) - (Get-Item $LockFile).LastWriteTime
    if ($lockAge.TotalMinutes -lt $LockTimeoutMinutes) {
        Write-Log "Another update is in progress (lock age: $($lockAge.TotalMinutes) min). Exiting."
        exit 0
    }
}
New-Item -ItemType File -Path $LockFile -Force | Out-Null

try {
    if (-not (Test-Path $Binary)) {
        Write-Log "Agent binary not found at $Binary; entering recovery mode"
        Recover-Agent
    } else {
        Write-Log "Running $Binary --update-check"
        $proc = Start-Process -FilePath $Binary -ArgumentList "--update-check" -Wait -PassThru -NoNewWindow
        Write-Log "emir-agent --update-check finished with exit code $($proc.ExitCode)"
        if ($proc.ExitCode -ne 0) {
            Write-Log "Update check failed; the running agent will retry on its next poll"
        }
    }
} catch {
    Write-Log "ERROR: $_"
    exit 1
} finally {
    Remove-Item -Path $LockFile -Force -ErrorAction SilentlyContinue
}
