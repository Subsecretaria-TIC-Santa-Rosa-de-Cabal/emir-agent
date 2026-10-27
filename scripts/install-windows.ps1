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
    [string]$CoreURL = "https://api.emir.gov.co",
    [string]$Version = "",
    [string]$Repo = "Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent",
    [string]$InstallDir = "C:\Program Files\emir-agent",
    [bool]$UpdateAtStartup = $true,
    [int]$UpdateBootDelaySeconds = 90,
    [switch]$SkipPairing
)

$ErrorActionPreference = "Stop"

# GitHub requires TLS 1.2; PowerShell 5.1 defaults to older protocols on some systems.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Error "This script must be run as Administrator."
    exit 1
}

function Get-LatestReleaseVersion {
    param([string]$Repository)
    $apiUrl = "https://api.github.com/repos/$Repository/releases/latest"
    try {
        $response = Invoke-WebRequest -Uri $apiUrl -UseBasicParsing -Headers @{
            'User-Agent' = 'emir-agent-installer'
        } | ConvertFrom-Json
        if (-not $response.tag_name) {
            throw "GitHub API did not return a tag_name"
        }
        return $response.tag_name.TrimStart('v')
    } catch {
        Write-Error "Failed to determine latest release version from GitHub: $_"
        exit 1
    }
}

if ([string]::IsNullOrWhiteSpace($Version)) {
    Write-Host "Version not specified. Looking up the latest release on GitHub..."
    $Version = Get-LatestReleaseVersion -Repository $Repo
    Write-Host "Latest release is v$Version"
}

# Normalize version so the URL always contains the 'v' prefix.
$TagVersion = $Version
if (-not $TagVersion.StartsWith('v')) {
    $TagVersion = "v$TagVersion"
}

$ArchiveName = "emir-agent-windows-amd64.zip"
$ServiceName = "emir-agent"
$AgentBinary = Join-Path $InstallDir "emir-agent.exe"
$PopupBinary = Join-Path $InstallDir "emir-agent-popup.exe"
$AssetsDir = Join-Path $InstallDir "assets"
$StateDir = Join-Path $env:PROGRAMDATA "emir-agent"
$StatePath = Join-Path $StateDir "state.json"

# Stop/remove any existing service so the binary can be replaced.
$ExistingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($ExistingService) {
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    sc.exe delete $ServiceName | Out-Null
    Start-Sleep -Seconds 2
}

# Remove any existing startup update task; it will be re-created below.
$UpdateTaskName = "EMIR Agent Update At Startup"
Unregister-ScheduledTask -TaskName $UpdateTaskName -Confirm:$false -ErrorAction SilentlyContinue

$TempDir = Join-Path $env:TEMP "emir-agent-install-$Version"
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

function Download-File {
    param(
        [string]$Url,
        [string]$OutFile
    )

    $maxAttempts = 3
    for ($attempt = 1; $attempt -le $maxAttempts; $attempt++) {
        try {
            Invoke-WebRequest -Uri $Url -OutFile $OutFile -UseBasicParsing -Headers @{
                'User-Agent' = 'emir-agent-installer'
            }
            return
        } catch {
            Write-Warning "Download attempt $attempt of $maxAttempts failed: $_"
            if ($attempt -eq $maxAttempts) {
                throw $_
            }
            Start-Sleep -Seconds 2
        }
    }
}

$ArchiveUrl = "https://github.com/$Repo/releases/download/$TagVersion/$ArchiveName"
$ArchivePath = Join-Path $TempDir $ArchiveName

try {
    Write-Host "Downloading $ArchiveName from GitHub ($ArchiveUrl)..."
    Download-File -Url $ArchiveUrl -OutFile $ArchivePath
} catch {
    Write-Warning "Invoke-WebRequest failed, trying curl.exe..."
    $curl = Get-Command curl.exe -ErrorAction SilentlyContinue
    if (-not $curl) {
        Write-Error "Neither Invoke-WebRequest nor curl.exe is available to download the agent."
        exit 1
    }

    & curl.exe -fsSL -A 'emir-agent-installer' -o $ArchivePath $ArchiveUrl
    if ($LASTEXITCODE -ne 0) {
        Write-Error "curl.exe failed to download the release archive."
        exit 1
    }
}

if (-not (Test-Path $ArchivePath)) {
    Write-Error "Downloaded archive was not found at $ArchivePath"
    exit 1
}

Write-Host "Extracting to $TempDir..."
Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null
Expand-Archive -Path $ArchivePath -DestinationPath $TempDir -Force

$ExtractedAgent = Join-Path $TempDir "emir-agent-windows-amd64.exe"
$ExtractedPopup = Join-Path $TempDir "emir-agent-popup-windows-amd64.exe"
$ExtractedAssets = Join-Path $TempDir "assets"

if (-not (Test-Path $ExtractedAgent)) {
    Write-Error "Agent binary was not found in the archive at $ExtractedAgent"
    exit 1
}
if (-not (Test-Path $ExtractedPopup)) {
    Write-Error "Popup binary was not found in the archive at $ExtractedPopup"
    exit 1
}

Write-Host "Installing to $InstallDir..."
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Move-Item -Path $ExtractedAgent -Destination $AgentBinary -Force -ErrorAction SilentlyContinue
Move-Item -Path $ExtractedPopup -Destination $PopupBinary -Force -ErrorAction SilentlyContinue
if (Test-Path $ExtractedAssets) {
    New-Item -ItemType Directory -Path $AssetsDir -Force | Out-Null
    $resolvedExtracted = Resolve-Path $ExtractedAssets
    Get-ChildItem -Path $resolvedExtracted -Recurse -File | ForEach-Object {
        $relativePath = $_.FullName.Substring($resolvedExtracted.Path.Length).TrimStart('\', '/')
        $dest = Join-Path $AssetsDir $relativePath
        $destDir = Split-Path -Parent $dest
        New-Item -ItemType Directory -Path $destDir -Force | Out-Null

        if ($_.FullName -eq $dest) {
            Write-Warning "Skipping copy of $($_.FullName) because source and destination are the same path."
        } else {
            Copy-Item -Path $_.FullName -Destination $dest -Force
        }
    }
}

# Copy the boot update helper and record install metadata for recovery.
$BootScript = Join-Path $PSScriptRoot "update-at-boot.ps1"
if (Test-Path $BootScript) {
    Copy-Item -Path $BootScript -Destination $InstallDir -Force
}

@(
    "REPO=$Repo"
    "VERSION=$Version"
    "ARCHIVE=$ArchiveName"
) | Set-Content -Path (Join-Path $InstallDir "install.conf") -Encoding ASCII

# Persist core URL as an environment variable for the service.
if ($CoreURL -ne "") {
    [Environment]::SetEnvironmentVariable("EMIR_CORE_URL", $CoreURL, "Machine")
    $env:EMIR_CORE_URL = $CoreURL
}

# Use a machine-wide state directory so the service (running as SYSTEM) can
# access the same state and credentials created during interactive pairing.
[Environment]::SetEnvironmentVariable("EMIR_STATE_PATH", $StatePath, "Machine")
$env:EMIR_STATE_PATH = $StatePath
New-Item -ItemType Directory -Path $StateDir -Force | Out-Null

function Grant-ReadAccessToAuthenticatedUsers {
    param(
        [string]$Path,
        [switch]$IsDirectory
    )
    $acl = Get-Acl -Path $Path
    $inheritanceFlags = if ($IsDirectory) { "ContainerInherit,ObjectInherit" } else { "None" }
    $rule = New-Object System.Security.AccessControl.FileSystemAccessRule(
        "Authenticated Users",
        "ReadAndExecute",
        $inheritanceFlags,
        "None",
        "Allow"
    )
    $acl.AddAccessRule($rule)
    Set-Acl -Path $Path -AclObject $acl
}

# Pair interactively before running as a service unless requested to skip.
if ($SkipPairing) {
    Write-Host "`nSkipping interactive pairing (--SkipPairing)."
} else {
    Write-Host "`nPairing the agent with emir-core..."
    & $AgentBinary --pair

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Pairing failed. The service will not be started."
        exit 1
    }
}

# Allow the logged-on user (and any future interactive user) to read the state
# and credentials so the popup process can authenticate when launched from the
# service into the user's session.
Grant-ReadAccessToAuthenticatedUsers -Path $StateDir -IsDirectory
$credentialsPath = Join-Path $StateDir "credentials.json"
if (Test-Path $credentialsPath) {
    Grant-ReadAccessToAuthenticatedUsers -Path $credentialsPath
}

Write-Host "Registering Windows service $ServiceName..."
sc.exe create $ServiceName binPath= "$AgentBinary" start= auto DisplayName= "EMIR Agent" | Out-Null
sc.exe failure $ServiceName reset= 86400 actions= restart/60000/restart/60000/restart/60000 | Out-Null

Start-Service -Name $ServiceName

# Register a startup task that checks for mandatory updates shortly after boot.
if ($UpdateAtStartup) {
    $UpdateTaskName = "EMIR Agent Update At Startup"
    $Action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$InstallDir\update-at-boot.ps1`""
    $Trigger = New-ScheduledTaskTrigger -AtStartup
    $Trigger.Delay = "PT$($UpdateBootDelaySeconds)S"
    $Principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
    $Settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -Hidden

    Register-ScheduledTask -TaskName $UpdateTaskName -Action $Action -Trigger $Trigger -Principal $Principal -Settings $Settings -Force | Out-Null
    Write-Host "Registered startup update task '$UpdateTaskName' (delay: ${UpdateBootDelaySeconds}s)."
}

Write-Host "emir-agent installed and running."
