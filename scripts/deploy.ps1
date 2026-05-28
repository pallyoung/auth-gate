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

# Copy files to dist
Write-Host "[2/3] Copying to dist..." -ForegroundColor Yellow
Copy-Item (Join-Path $ProjectRoot "packages\server\bin\auth-gate.exe") -Destination $DistDir -Force
Copy-Item (Join-Path $ProjectRoot "packages\server\configs\config.yaml") -Destination $DistDir -Force

$srcWeb = Join-Path $ProjectRoot "packages\web\dist"
$dstWeb = Join-Path $DistDir "web"
Remove-Item $dstWeb -Recurse -Force -ErrorAction SilentlyContinue
Copy-Item $srcWeb -Destination $dstWeb -Recurse -Force

# Start service
Write-Host "[3/3] Starting service..." -ForegroundColor Yellow
Set-Location $DistDir

$proc = Start-Process -FilePath ".\auth-gate.exe" -PassThru -NoNewWindow
if ($proc) {
    Write-Host ""
    Write-Host "=== Deploy complete ===" -ForegroundColor Green
    Write-Host "Control plane: http://localhost:8080/_authgate" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Press Ctrl+C to stop" -ForegroundColor Yellow
    
    $proc.WaitForExit()
}
