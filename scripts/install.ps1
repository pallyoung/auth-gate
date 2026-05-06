# Auth Gate Install Script for Windows

$ErrorActionPreference = "Stop"

# Get project root
if ($PSScriptRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
} else {
    $ProjectRoot = $PSCommandPath | Split-Path | Split-Path
}

Write-Host "=== Auth Gate Install ===" -ForegroundColor Cyan
Write-Host "Project: $ProjectRoot" -ForegroundColor Gray

# Check Go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go is not installed" -ForegroundColor Red
    exit 1
}
Write-Host "Go: $(go version)" -ForegroundColor Green

# Check Node
if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Node.js is not installed" -ForegroundColor Red
    exit 1
}
Write-Host "Node: $(node --version), npm: $(npm --version)" -ForegroundColor Green

# Paths
$BinDir = Join-Path $ProjectRoot "packages\server\bin"
$WebDir = Join-Path $ProjectRoot "packages\web"
$ServerDir = Join-Path $ProjectRoot "packages\server"

Write-Host ""
Write-Host "[1/4] Installing frontend dependencies..." -ForegroundColor Yellow
Set-Location $WebDir
npm install --include=dev --legacy-peer-deps

Write-Host "[2/4] Building frontend..." -ForegroundColor Yellow
npm run build

Write-Host "[3/4] Building server..." -ForegroundColor Yellow
Set-Location $ServerDir

# Clean old binary first
$exePath = Join-Path $BinDir "auth-gate.exe"
if (Test-Path $exePath) {
    Write-Host "Removing old binary..." -ForegroundColor Gray
    Remove-Item $exePath -Force -ErrorAction SilentlyContinue
    Start-Sleep -Milliseconds 500
}

& go build -ldflags="-s -w" -o bin\auth-gate.exe .\cmd\server

Set-Location $ProjectRoot
Write-Host "[4/4] Done!" -ForegroundColor Green
Write-Host ""
Write-Host "Binary: $exePath" -ForegroundColor Cyan
