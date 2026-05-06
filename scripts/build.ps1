# Auth Gate Build Script for Windows

$ErrorActionPreference = "Stop"

# Get project root
if ($PSScriptRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
} else {
    $ProjectRoot = $PSCommandPath | Split-Path | Split-Path
}

Write-Host "=== Auth Gate Build ===" -ForegroundColor Cyan

# Build frontend
Write-Host "[1/2] Building frontend..." -ForegroundColor Yellow
$webDir = Join-Path $ProjectRoot "packages\web"
Set-Location $webDir
npm install --legacy-peer-deps
npm run build

# Build server
Write-Host "[2/2] Building server..." -ForegroundColor Yellow
$serverDir = Join-Path $ProjectRoot "packages\server"
Set-Location $serverDir
go build -o bin\auth-gate.exe .\cmd\server

Set-Location $ProjectRoot
Write-Host "Build complete: packages\server\bin\auth-gate.exe" -ForegroundColor Green
