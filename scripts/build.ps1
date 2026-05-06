# Auth Gate Build Script for Windows

$ErrorActionPreference = "Stop"

# Get project root
if ($PSScriptRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
} else {
    $ProjectRoot = $PSCommandPath | Split-Path | Split-Path
}

Write-Host "=== Auth Gate Build ===" -ForegroundColor Cyan
Write-Host "Project: $ProjectRoot" -ForegroundColor Gray

# Check Go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go is not installed" -ForegroundColor Red
    exit 1
}
Write-Host "Go: $(go version)" -ForegroundColor Green

# Paths
$WebDir = Join-Path $ProjectRoot "packages\web"
$ServerDir = Join-Path $ProjectRoot "packages\server"
$BinDir = Join-Path $ServerDir "bin"
$ExePath = Join-Path $BinDir "auth-gate.exe"

Write-Host ""
Write-Host "[1/2] Building frontend..." -ForegroundColor Yellow
Set-Location $WebDir
npm install --legacy-peer-deps
npm run build

Write-Host "[2/2] Building server..." -ForegroundColor Yellow
Set-Location $ServerDir

if (Test-Path $ExePath) {
    Write-Host "Removing old binary..." -ForegroundColor Gray
    Remove-Item $ExePath -Force -ErrorAction SilentlyContinue
    Start-Sleep -Milliseconds 500
}

& go build -ldflags="-s -w" -o bin\auth-gate.exe .\cmd\server

if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

Set-Location $ProjectRoot
Write-Host ""
Write-Host "Build complete: $ExePath" -ForegroundColor Green
