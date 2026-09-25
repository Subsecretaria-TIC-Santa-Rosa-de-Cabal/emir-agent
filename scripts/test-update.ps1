#Requires -Version 5.1
<#
.SYNOPSIS
    Local development test for emir-agent auto-update on Windows.
.DESCRIPTION
    Builds an "old" and a "new" agent binary, starts a local HTTP server to
    serve the new binary, and prints the SQL needed to register the fake
    release in emir-core. Then runs the old agent in console mode so you can
    observe the update flow.
#>
param(
    [string]$CoreURL = "http://localhost:8000",
    [string]$OldVersion = "0.3.0",
    [string]$NewVersion = "0.3.5"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $PSScriptRoot
$DistDir = Join-Path $RepoRoot "dist-test"
$ServerPort = 9999

New-Item -ItemType Directory -Path $DistDir -Force | Out-Null

Write-Host "Building old agent version $OldVersion..."
$OldBinary = Join-Path $DistDir "emir-agent-old.exe"
go build -ldflags "-s -w -X github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models.Version=$OldVersion" -o $OldBinary $RepoRoot

Write-Host "Building new agent version $NewVersion..."
$NewBinary = Join-Path $DistDir "emir-agent-new.exe"
go build -ldflags "-s -w -X github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models.Version=$NewVersion" -o $NewBinary $RepoRoot

$Checksum = (Get-FileHash $NewBinary -Algorithm SHA256).Hash.ToLower()
Write-Host "New binary checksum: $Checksum"

Write-Host "Starting local HTTP server on port $ServerPort..."
$ServerJob = Start-Job {
    param($Dir, $Port)
    $listener = New-Object System.Net.HttpListener
    $listener.Prefixes.Add("http://+:$Port/")
    $listener.Start()
    while ($listener.IsListening) {
        $context = $listener.GetContext()
        $path = Join-Path $Dir ($context.Request.Url.AbsolutePath.TrimStart('/'))
        if (Test-Path $path) {
            $bytes = [System.IO.File]::ReadAllBytes($path)
            $context.Response.StatusCode = 200
            $context.Response.OutputStream.Write($bytes, 0, $bytes.Length)
        } else {
            $context.Response.StatusCode = 404
        }
        $context.Response.Close()
    }
} -ArgumentList $DistDir, $ServerPort

Start-Sleep -Seconds 2

$DownloadURL = "http://localhost:$ServerPort/emir-agent-new.exe"

Write-Host ""
Write-Host "=== SQL to register the fake release in emir-core ===" -ForegroundColor Cyan
Write-Host @"
INSERT INTO agent_releases (
    id, version, download_url, checksum, is_mandatory, release_notes, enabled, active, registration_date, last_update
) VALUES (
    gen_random_uuid(),
    '$NewVersion',
    '$DownloadURL',
    '$Checksum',
    true,
    'Local test update',
    true,
    true,
    now(),
    now()
);
"@
Write-Host ""

Write-Host "=== Next steps ===" -ForegroundColor Cyan
Write-Host "1. Run the SQL above in your emir-core database."
Write-Host "2. Ensure emir-core is running at $CoreURL."
Write-Host "3. Pair the old agent if needed:"
Write-Host "   $OldBinary --pair"
Write-Host "4. Run the old agent and watch for the update:"
Write-Host "   `$env:EMIR_CORE_URL='$CoreURL'; $OldBinary"
Write-Host "5. After the update, verify the running binary version:"
Write-Host "   (Get-Item '$NewBinary').VersionInfo.FileVersion"
Write-Host ""

# Keep server alive until user presses Enter.
Read-Host "Press Enter to stop the local HTTP server"
Stop-Job $ServerJob
Remove-Job $ServerJob
