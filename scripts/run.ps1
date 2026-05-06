# Auth Gate Run Script for Windows

$ProjectRoot = Split-Path -Parent $PSScriptRoot

# Build if needed
if (-not (Test-Path "$ProjectRoot\packages\server\bin\auth-gate.exe")) {
    Write-Host "Binary not found, building first..." -ForegroundColor Yellow
    & "$ProjectRoot\scripts\build.ps1"
}

# Run
Push-Location "$ProjectRoot\packages\server"
.\bin\auth-gate.exe
Pop-Location
