# Docker Build Script for Windows

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot

Write-Host "=== Docker Build ===" -ForegroundColor Cyan

docker build -t auth-gate:latest $ProjectRoot

Write-Host "Image: auth-gate:latest" -ForegroundColor Green
