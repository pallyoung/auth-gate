# Auth Gate Install Script for Windows

$ErrorActionPreference = "Stop"

Write-Host "=== Auth Gate Install ===" -ForegroundColor Cyan

# Check Go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go is not installed" -ForegroundColor Red
    exit 1
}

# Check Node
if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Node.js is not installed" -ForegroundColor Red
    exit 1
}

$ProjectRoot = Split-Path -Parent $PSScriptRoot

# Install frontend deps
Write-Host "[1/4] Installing frontend dependencies..." -ForegroundColor Yellow
Push-Location "$ProjectRoot\packages\web"
npm install
Pop-Location

# Build frontend
Write-Host "[2/4] Building frontend..." -ForegroundColor Yellow
Push-Location "$ProjectRoot\packages\web"
npm run build
Pop-Location

# Build server
Write-Host "[3/4] Building server..." -ForegroundColor Yellow
Push-Location "$ProjectRoot\packages\server"
go build -o bin\auth-gate.exe .\cmd\server
Pop-Location

Write-Host "[4/4] Build complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Binary: packages\server\bin\auth-gate.exe" -ForegroundColor Cyan
Write-Host ""
Write-Host "Run with: .\scripts\run.ps1" -ForegroundColor Green
