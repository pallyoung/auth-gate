# Docker Run Script for Windows

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot

Write-Host "=== Docker Run ===" -ForegroundColor Cyan

# Remove old container if exists
$existing = docker ps -a --format "{{.Names}}" | Where-Object { $_ -eq "auth-gate" }
if ($existing) {
    docker stop auth-gate | Out-Null
    docker rm auth-gate | Out-Null
}

# Start new container
docker run -d `
    --name auth-gate `
    -p 8080:8080 `
    -v auth-gate-data:/app/data `
    -v "$ProjectRoot\packages\server\configs\config.yaml:C:\app\config.yaml:ro" `
    --restart unless-stopped `
    auth-gate:latest

Write-Host "Container started" -ForegroundColor Green
docker logs auth-gate
