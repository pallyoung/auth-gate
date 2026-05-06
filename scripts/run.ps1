# Auth Gate Run Script for Windows

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot

# Install deps if needed
if (-not (Test-Path "$ProjectRoot\packages\web\node_modules")) {
    Write-Host "Installing frontend dependencies..." -ForegroundColor Yellow
    Push-Location "$ProjectRoot\packages\web"
    npm install
    Pop-Location
}

# Build if needed
if (-not (Test-Path "$ProjectRoot\packages\server\bin\auth-gate.exe")) {
    Write-Host "Building..." -ForegroundColor Yellow
    Push-Location "$ProjectRoot\packages\web"
    npm run build
    Pop-Location
    Push-Location "$ProjectRoot\packages\server"
    go build -o bin\auth-gate.exe .\cmd\server
    Pop-Location
}

# Run
Push-Location "$ProjectRoot\packages\server"
.\bin\auth-gate.exe
Pop-Location
