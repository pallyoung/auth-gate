# Auth Gate Run Script for Windows

$ErrorActionPreference = "Stop"

# Get project root
if ($PSScriptRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
} else {
    $ProjectRoot = $PSCommandPath | Split-Path | Split-Path
}

$DistDir = Join-Path $ProjectRoot "dist"
$ExePath = Join-Path $DistDir "auth-gate.exe"

# Build if needed
if (-not (Test-Path $ExePath)) {
    Write-Host "Distribution not found, deploying..." -ForegroundColor Yellow
    & "$ProjectRoot\scripts\deploy.ps1"
    return
}

# Start service
Set-Location $DistDir
Write-Host "Starting Auth Gate on http://localhost:8080" -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop" -ForegroundColor Gray
Write-Host ""
.\auth-gate.exe
