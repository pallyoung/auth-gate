# Auth Gate Release Build Script for Windows

$ErrorActionPreference = "Stop"

# Get project root
if ($PSScriptRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
} else {
    $ProjectRoot = $PSCommandPath | Split-Path | Split-Path
}

$Version = if ($args[0]) { $args[0] } else { "latest" }
$OutputDir = Join-Path $ProjectRoot "dist\release"

Write-Host "=== Building Release $Version ===" -ForegroundColor Cyan

# Check Go
$goCmd = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCmd) {
    Write-Host "Error: Go is not installed" -ForegroundColor Red
    exit 1
}

# Create output dir
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

# Build frontend
Write-Host "[1/4] Building frontend..." -ForegroundColor Yellow
Set-Location (Join-Path $ProjectRoot "packages\web")
npm install
npm run build

# Build Linux binaries
Write-Host "[2/4] Building Linux..." -ForegroundColor Yellow
Set-Location (Join-Path $ProjectRoot "packages\server")
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
& go build -ldflags="-s -w" -o "$OutputDir\auth-gate-linux-amd64" .\cmd\server
$env:GOARCH = "arm64"
& go build -ldflags="-s -w" -o "$OutputDir\auth-gate-linux-arm64" .\cmd\server

# Build macOS binaries
Write-Host "[3/4] Building macOS..." -ForegroundColor Yellow
$env:GOOS = "darwin"
$env:GOARCH = "amd64"
& go build -ldflags="-s -w" -o "$OutputDir\auth-gate-darwin-amd64" .\cmd\server
$env:GOARCH = "arm64"
& go build -ldflags="-s -w" -o "$OutputDir\auth-gate-darwin-arm64" .\cmd\server

# Build Windows binary
Write-Host "[4/4] Building Windows..." -ForegroundColor Yellow
$env:GOOS = "windows"
$env:GOARCH = "amd64"
& go build -ldflags="-s -w" -o "$OutputDir\auth-gate-windows-amd64.exe" .\cmd\server

# Create zip packages
Write-Host "Packaging..." -ForegroundColor Yellow
Set-Location $OutputDir
Compress-Archive -Path "auth-gate-windows-amd64.exe" -DestinationPath "auth-gate-windows-amd64.zip" -Force

Write-Host "=== Release built in $OutputDir ===" -ForegroundColor Green
Get-ChildItem $OutputDir | Format-Table Name, Length -AutoSize
