# Auth Gate Deploy Script for Windows

$ErrorActionPreference = "Stop"

# Get project root
if ($PSScriptRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
} else {
    $ProjectRoot = $PSCommandPath | Split-Path | Split-Path
}

Write-Host "=== Auth Gate Deploy (Windows) ===" -ForegroundColor Cyan

# Build
Write-Host "[1/3] Building..." -ForegroundColor Yellow
& "$ProjectRoot\scripts\build.ps1"

# Determine install directory (use user dir if no admin rights)
$installDir = Join-Path $env:LOCALAPPDATA "AuthGate"
$configDir = Join-Path $env:LOCALAPPDATA "AuthGate"

Write-Host ""
Write-Host "Installing to: $installDir" -ForegroundColor Gray
Write-Host ""

# Stop existing service
Write-Host "[2/3] Stopping existing service..." -ForegroundColor Yellow
$serviceName = "AuthGate"
$svc = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($svc) {
    Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
    sc.exe delete $serviceName 2>$null
}

# Install binary
Write-Host "[3/3] Installing..." -ForegroundColor Yellow
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
$srcBin = Join-Path $ProjectRoot "packages\server\bin\auth-gate.exe"
$dstBin = Join-Path $installDir "auth-gate.exe"

if (Test-Path $dstBin) {
    Write-Host "Removing old binary..." -ForegroundColor Gray
    Remove-Item $dstBin -Force -ErrorAction SilentlyContinue
    Start-Sleep -Milliseconds 300
}

Copy-Item $srcBin -Destination $installDir -Force

# Copy config
New-Item -ItemType Directory -Force -Path $configDir | Out-Null
$srcConfig = Join-Path $ProjectRoot "packages\server\configs\config.yaml"
$dstConfig = Join-Path $configDir "config.yaml"
if (-not (Test-Path $dstConfig)) {
    Copy-Item $srcConfig -Destination $configDir -Force
}

Write-Host ""
Write-Host "=== Deploy complete ===" -ForegroundColor Green
Write-Host ""
Write-Host "Binary: $dstBin" -ForegroundColor Cyan
Write-Host "Config: $dstConfig" -ForegroundColor Cyan
Write-Host ""
Write-Host "Run with: & '$dstBin'" -ForegroundColor Green
