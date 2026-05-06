# Auth Gate Deploy Script for Windows

$ErrorActionPreference = "Stop"

# Get project root
if ($PSScriptRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
} else {
    $ProjectRoot = $PSCommandPath | Split-Path | Split-Path
}

Write-Host "=== Auth Gate Deploy (Windows) ===" -ForegroundColor Cyan

# Build
Write-Host "[1/3] Building..." -ForegroundColor Yellow
& "$ProjectRoot\scripts\build.ps1"

# Create dist directory
$DistDir = Join-Path $ProjectRoot "dist"
New-Item -ItemType Directory -Force -Path $DistDir | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $DistDir "web") | Out-Null

# Stop existing service
Write-Host "[2/3] Stopping existing service..." -ForegroundColor Yellow
$serviceName = "AuthGate"
$svc = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($svc) {
    Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
    sc.exe delete $serviceName 2>$null
}

# Copy files to dist
Write-Host "[3/3] Copying to dist..." -ForegroundColor Yellow
Copy-Item (Join-Path $ProjectRoot "packages\server\bin\auth-gate.exe") -Destination $DistDir -Force
Copy-Item (Join-Path $ProjectRoot "packages\server\configs\config.yaml") -Destination $DistDir -Force

# Copy web dist
$srcWeb = Join-Path $ProjectRoot "packages\web\dist"
$dstWeb = Join-Path $DistDir "web"
Copy-Item $srcWeb -Destination $dstWeb -Recurse -Force

Write-Host ""
Write-Host "=== Deploy complete ===" -ForegroundColor Green
Write-Host ""
Write-Host "Run: cd dist; .\auth-gate.exe" -ForegroundColor Cyan
Write-Host "Then visit: http://localhost:8080" -ForegroundColor Cyan
