# Auth Gate Install Script for Windows

$ErrorActionPreference = "Stop"

# Get project root
if ($PSScriptRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
} else {
    $ProjectRoot = $PSCommandPath | Split-Path | Split-Path
}

Write-Host "=== Auth Gate Install ===" -ForegroundColor Cyan

# Check Go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go is not installed" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please install Go first:" -ForegroundColor Yellow
    Write-Host "  https://go.dev/dl/" -ForegroundColor White
    Write-Host ""
    Write-Host "After installation, restart PowerShell and try again." -ForegroundColor Yellow
    exit 1
}

Write-Host "Go found: $(go version)" -ForegroundColor Green

# Check Node
if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Node.js is not installed" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please install Node.js first:" -ForegroundColor Yellow
    Write-Host "  https://nodejs.org/" -ForegroundColor White
    exit 1
}

Write-Host "Node found: $(node --version)" -ForegroundColor Green
Write-Host "npm found: $(npm --version)" -ForegroundColor Green

# Use user-writable directories to avoid permission issues
$BinDir = Join-Path $ProjectRoot "packages\server\bin"
$WebDistDir = Join-Path $ProjectRoot "packages\web\dist"

# Ensure directories exist and are writable
Write-Host "Preparing directories..." -ForegroundColor Yellow
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

# Install frontend deps
Write-Host "[1/4] Installing frontend dependencies..." -ForegroundColor Yellow
Set-Location (Join-Path $ProjectRoot "packages\web")
npm install --legacy-peer-deps

# Build frontend
Write-Host "[2/4] Building frontend..." -ForegroundColor Yellow
npm run build

# Build server
Write-Host "[3/4] Building server..." -ForegroundColor Yellow
Set-Location (Join-Path $ProjectRoot "packages\server")
& go build -ldflags="-s -w" -o bin\auth-gate.exe .\cmd\server

Set-Location $ProjectRoot
Write-Host "[4/4] Build complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Binary: packages\server\bin\auth-gate.exe" -ForegroundColor Cyan
Write-Host ""
Write-Host "Run with: .\scripts\run.ps1" -ForegroundColor Green
