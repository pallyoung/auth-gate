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
$srcBin = Join-Path $ProjectRoot "packages\server\bin\auth-gate.exe"
$dstBin = Join-Path $DistDir "auth-gate.exe"
$srcConfig = Join-Path $ProjectRoot "packages\server\configs\config.yaml"
$dstConfig = Join-Path $DistDir "config.yaml"
$srcWebDist = Join-Path $ProjectRoot "packages\web\dist"

if (Test-Path $dstBin) {
    Remove-Item $dstBin -Force -ErrorAction SilentlyContinue
    Start-Sleep -Milliseconds 200
}

Copy-Item $srcBin -Destination $DistDir -Force
Copy-Item $srcConfig -Destination $DistDir -Force

# Copy web dist
if (Test-Path $srcWebDist) {
    $dstWebDist = Join-Path $DistDir "web"
    if (Test-Path $dstWebDist) {
        Remove-Item $dstWebDist -Recurse -Force -ErrorAction SilentlyContinue
    }
    Copy-Item $srcWebDist -Destination $dstWebDist -Recurse -Force
}

Write-Host ""
Write-Host "=== Deploy complete ===" -ForegroundColor Green
Write-Host ""
Write-Host "Files in dist:" -ForegroundColor Cyan
Get-ChildItem $DistDir -Recurse | Select-Object FullName
Write-Host ""
Write-Host "Run with: & '$dstBin'" -ForegroundColor Green
