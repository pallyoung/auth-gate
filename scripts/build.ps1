# Auth Gate Build Script for Windows

$ErrorActionPreference = "Stop"

Write-Host "=== Auth Gate Build ===" -ForegroundColor Cyan

$ProjectRoot = Split-Path -Parent $PSScriptRoot

# Build frontend
Write-Host "[1/2] Building frontend..." -ForegroundColor Yellow
Push-Location "$ProjectRoot\packages\web"
npm install
npm run build
Pop-Location

# Build server
Write-Host "[2/2] Building server..." -ForegroundColor Yellow
Push-Location "$ProjectRoot\packages\server"
go build -o bin\auth-gate.exe .\cmd\server
Pop-Location

Write-Host "Build complete: packages\server\bin\auth-gate.exe" -ForegroundColor Green
