# Auth Gate Run Script for Windows

$ErrorActionPreference = "Stop"

# Get project root
if ($PSScriptRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
} else {
    $ProjectRoot = $PSCommandPath | Split-Path | Split-Path
}

# Install deps if needed
$webDir = Join-Path $ProjectRoot "packages\web"
if (-not (Test-Path (Join-Path $webDir "node_modules"))) {
    Write-Host "Installing frontend dependencies..." -ForegroundColor Yellow
    Set-Location $webDir
    npm install --legacy-peer-deps
}

# Build if needed
$serverBin = Join-Path $ProjectRoot "packages\server\bin\auth-gate.exe"
if (-not (Test-Path $serverBin)) {
    Write-Host "Building..." -ForegroundColor Yellow
    Set-Location $webDir
    npm run build
    
    Set-Location (Join-Path $ProjectRoot "packages\server")
    go build -o bin\auth-gate.exe .\cmd\server
}

# Run
Set-Location (Join-Path $ProjectRoot "packages\server")
.\bin\auth-gate.exe
