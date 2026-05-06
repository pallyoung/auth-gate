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
$goCmd = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCmd) {
    Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
    Write-Host "Please install Go from: https://go.dev/dl/" -ForegroundColor Yellow
    exit 1
}

# Check Node
$nodeCmd = Get-Command node -ErrorAction SilentlyContinue
if (-not $nodeCmd) {
    Write-Host "Error: Node.js is not installed or not in PATH" -ForegroundColor Red
    Write-Host "Please install Node.js from: https://nodejs.org/" -ForegroundColor Yellow
    exit 1
}

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
& go build -o bin\auth-gate.exe .\cmd\server

Set-Location $ProjectRoot
Write-Host "[4/4] Build complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Binary: packages\server\bin\auth-gate.exe" -ForegroundColor Cyan
Write-Host ""
Write-Host "Run with: .\scripts\run.ps1" -ForegroundColor Green
