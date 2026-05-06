# Docker Deploy Script for Windows

$ErrorActionPreference = "Stop"

# Get project root - handle both direct run and Makefile invocation
if ($PSScriptRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
} else {
    $ProjectRoot = $PSCommandPath | Split-Path | Split-Path
}

Write-Host "=== Docker Deploy ===" -ForegroundColor Cyan

# Stop and remove old container
Write-Host "[1/4] Removing old container..." -ForegroundColor Yellow
docker stop auth-gate 2>$null | Out-Null
docker rm auth-gate 2>$null | Out-Null

# Build image
Write-Host "[2/4] Building image..." -ForegroundColor Yellow
docker build -t auth-gate:latest $ProjectRoot

# Start container
Write-Host "[3/4] Starting container..." -ForegroundColor Yellow
$configPath = "$ProjectRoot\packages\server\configs\config.yaml"
docker run -d `
    --name auth-gate `
    -p 8080:8080 `
    -v auth-gate-data:/app/data `
    -v "${configPath}:C:\app\config.yaml:ro" `
    --restart unless-stopped `
    auth-gate:latest

# Cleanup unused images
Write-Host "[4/4] Cleanup..." -ForegroundColor Yellow
docker image prune -f | Out-Null

Write-Host "=== Deploy complete ===" -ForegroundColor Green
docker ps --filter "name=auth-gate"
