# Docker Run Script for Windows

$ErrorActionPreference = "Stop"

# Get project root
if ($PSScriptRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
} else {
    $ProjectRoot = $PSCommandPath | Split-Path | Split-Path
}

Write-Host "=== Docker Run ===" -ForegroundColor Cyan

# Remove old container if exists
$existing = docker ps -a --format "{{.Names}}" | Where-Object { $_ -eq "auth-gate" }
if ($existing) {
    docker stop auth-gate | Out-Null
    docker rm auth-gate | Out-Null
}

# Start new container
$configPath = "$ProjectRoot\packages\server\configs\config.yaml"
docker run -d `
    --name auth-gate `
    -p 8080:8080 `
    -v auth-gate-data:/app/data `
    -v "${configPath}:C:\app\config.yaml:ro" `
    --restart unless-stopped `
    auth-gate:latest

Write-Host "Container started" -ForegroundColor Green
docker logs auth-gate
