#Requires -Version 5.1
<#
.SYNOPSIS
    Builds and runs the popup binary locally with a test announcement.
.DESCRIPTION
    Useful for previewing the popup design without publishing a release or
    waiting for the agent cycle. The popup is shown without reporting an
    impression if no agent credentials are found.
.PARAMETER ImageUrl
    URL of the image to show in the popup.
#>
param(
    [string]$ImageUrl = "https://via.placeholder.com/1920x1080.png/3366cc/ffffff?text=Test+Image"
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$DistDir = Join-Path $env:TEMP "emir-agent-popup-test"
$AssetsDir = Join-Path $DistDir "assets"

New-Item -ItemType Directory -Path $DistDir -Force | Out-Null
New-Item -ItemType Directory -Path $AssetsDir -Force | Out-Null

$Env:CGO_ENABLED = "0"
$Env:GOTOOLCHAIN = "local"

Write-Host "Building popup binary..."
go build -o "$DistDir\emir-agent-popup.exe" "$ProjectRoot\cmd\popup"

Write-Host "Copying logo..."
Copy-Item -Path "$ProjectRoot\assets\logo.png" -Destination "$AssetsDir\logo.png" -Force

# Build strings using [char] so the source file encoding does not matter.
$InvertedExclamation = [char]0x00A1
$Aacute              = [char]0x00E1

$Announcement = @{
    identifier         = [Guid]::NewGuid().ToString()
    title              = "$InvertedExclamation`Explora nuestras nuevas funcionalidades!"
    description        = "Hemos lanzado nuevas herramientas que te permitir$Aacute`n ser m$Aacute`s productivo, trabajar de forma m$Aacute`s eficiente y disfrutar de una mejor experiencia en la plataforma."
    link_url           = "https://emir.gov.co"
    image_download_url = $ImageUrl
} | ConvertTo-Json

$JsonPath = Join-Path $DistDir "announcement.json"
$Utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText($JsonPath, $Announcement, $Utf8NoBom)

Write-Host "Starting popup..."
& "$DistDir\emir-agent-popup.exe" --file $JsonPath
