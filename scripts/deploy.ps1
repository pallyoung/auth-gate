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

# Stop existing service
Write-Host "[2/3] Stopping existing service..." -ForegroundColor Yellow
$serviceName = "AuthGate"
$svc = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($svc) {
    Stop-Service -Name $serviceName -Force
    sc.exe delete $serviceName
}

# Install binary to Program Files
Write-Host "[3/3] Installing..." -ForegroundColor Yellow
$installDir = "$env:PROGRAMFILES\AuthGate"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
Copy-Item "$ProjectRoot\packages\server\bin\auth-gate.exe" "$installDir\auth-gate.exe"

# Copy config
$configDir = "$env:PROGRAMDATA\AuthGate"
New-Item -ItemType Directory -Force -Path $configDir | Out-Null
if (-not (Test-Path "$configDir\config.yaml")) {
    Copy-Item "$ProjectRoot\packages\server\configs\config.yaml" "$configDir\config.yaml"
}

# Register as Windows Service
Write-Host "Registering service..." -ForegroundColor Yellow
sc.exe create AuthGate binPath= "$installDir\auth-gate.exe" start= auto DisplayName= "Auth Gate API Gateway" 2>$null
sc.exe config AuthGate obj= "NT AUTHORITY\LocalService" 2>$null

# Start service
Start-Service -Name AuthGate -ErrorAction SilentlyContinue
if (-not $?) {
    # Fallback: run directly if service failed
    Write-Host "Service registration failed, starting directly..." -ForegroundColor Yellow
    Start-Process "$installDir\auth-gate.exe"
}

Write-Host "=== Deploy complete ===" -ForegroundColor Green
Get-Service -Name AuthGate -ErrorAction SilentlyContinue | Format-Table Name, Status, DisplayName -AutoSize
