# Auth Gate Dev Script (Windows)
# Starts Go backend + Vite dev server together.
# Press Ctrl+C to stop both.

$ErrorActionPreference = "Continue"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Write-Host "=== Auth Gate Dev ===" -ForegroundColor Cyan
Write-Host "Project root: $ProjectRoot" -ForegroundColor Gray
Write-Host ""

# Check prerequisites
Write-Host "[0/3] Checking prerequisites..." -ForegroundColor Yellow

$goCmd = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCmd) {
    Write-Host "  Error: go not found in PATH" -ForegroundColor Red
    exit 1
}
Write-Host "  go:  $($goCmd.Source)" -ForegroundColor Gray

$npmCmd = Get-Command npm -ErrorAction SilentlyContinue
if (-not $npmCmd) {
    Write-Host "  Error: npm not found in PATH" -ForegroundColor Red
    exit 1
}
Write-Host "  npm: $($npmCmd.Source)" -ForegroundColor Gray
Write-Host ""

# Install web dependencies if needed
$webDir = Join-Path $ProjectRoot "packages\web"
$nodeModules = Join-Path $webDir "node_modules"
if (-not (Test-Path $nodeModules)) {
    Write-Host "[1/3] Installing web dependencies..." -ForegroundColor Yellow
    Push-Location $webDir
    npm install
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  Error: npm install failed (exit code $LASTEXITCODE)" -ForegroundColor Red
        Pop-Location
        exit 1
    }
    Pop-Location
    Write-Host "  Done." -ForegroundColor Gray
} else {
    Write-Host "[1/3] Web dependencies already installed." -ForegroundColor Gray
}

Write-Host "[2/3] Starting Go backend (admin :9000, proxy :80)..." -ForegroundColor Yellow

# Start Go backend in background via a separate PowerShell process.
# Avoid Start-Process -Environment which replaces all env vars in PS 5.1.
$serverDir = Join-Path $ProjectRoot "packages\server"
$goLog = Join-Path $ProjectRoot "dev-server.log"

$goProcess = Start-Process -FilePath "powershell.exe" -ArgumentList @(
    "-NoProfile",
    "-Command",
    "`$env:DEBUG='true'; Set-Location '$serverDir'; air 2>&1 | Tee-Object -FilePath '$goLog'"
) -PassThru -WindowStyle Hidden

if (-not $goProcess) {
    Write-Host "  Error: failed to start Go backend" -ForegroundColor Red
    exit 1
}
Write-Host "  Go backend started (PID $($goProcess.Id))" -ForegroundColor Gray
Write-Host "  Log file: $goLog" -ForegroundColor Gray

# Give the backend a moment to start
Start-Sleep -Seconds 2

if ($goProcess.HasExited) {
    Write-Host "  Error: Go backend exited immediately (code $($goProcess.ExitCode))" -ForegroundColor Red
    if (Test-Path $goLog) {
        Write-Host "  Log output:" -ForegroundColor Red
        Get-Content $goLog | ForEach-Object { Write-Host "    $_" -ForegroundColor Red }
    }
    exit 1
}

Write-Host "[3/3] Starting Vite dev server (port 5174)..." -ForegroundColor Yellow
Write-Host ""
Write-Host "  Admin UI:  http://localhost:5174" -ForegroundColor Gray
Write-Host "  Admin API: http://127.0.0.1:9000/api" -ForegroundColor Gray
Write-Host "  Proxy:     http://localhost:80" -ForegroundColor Gray
Write-Host ""
Write-Host "Press Ctrl+C to stop" -ForegroundColor Gray
Write-Host ""

try {
    Push-Location $webDir
    npm run dev
} finally {
    Pop-Location

    # Kill the air parent process
    if ($goProcess -and -not $goProcess.HasExited) {
        Write-Host ""
        Write-Host "Stopping Go backend..." -ForegroundColor Yellow
        Stop-Process -Id $goProcess.Id -Force -ErrorAction SilentlyContinue
    }

    # Kill ALL auth-gate.exe and air child processes (air spawns children
    # that survive when the parent is killed).
    Get-Process -Name "auth-gate" -ErrorAction SilentlyContinue |
        Stop-Process -Force -ErrorAction SilentlyContinue
    Get-Process -Name "air" -ErrorAction SilentlyContinue |
        Stop-Process -Force -ErrorAction SilentlyContinue

    Write-Host "Dev environment stopped." -ForegroundColor Cyan
}
