# Docker Build Script for Windows

$ErrorActionPreference = "Stop"

# Get project root
if ($PSScriptRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
} else {
    $ProjectRoot = $PSCommandPath | Split-Path | Split-Path
}

Write-Host "=== Docker Build ===" -ForegroundColor Cyan

docker build -t auth-gate:latest $ProjectRoot

Write-Host "Image: auth-gate:latest" -ForegroundColor Green
