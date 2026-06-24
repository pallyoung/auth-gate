# Auth Gate Installer for Windows
# Usage: .\install.ps1

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\auth-gate"
)

$ErrorActionPreference = "Stop"

Write-Host "=== Installing Auth Gate ===" -ForegroundColor Green

# Create install directory
Write-Host "Installing to $InstallDir..."
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# Copy files
Write-Host "Copying files..."
Copy-Item -Path ".\auth-gate.exe" -Destination "$InstallDir\" -Force
Copy-Item -Path ".\web" -Destination "$InstallDir\" -Recurse -Force

# Create config if not exists
if (-not (Test-Path "$InstallDir\config.yaml")) {
    if (Test-Path "$InstallDir\config.yaml.example") {
        Copy-Item -Path "$InstallDir\config.yaml.example" -Destination "$InstallDir\config.yaml"
        Write-Host "Created config.yaml from example. Please edit it." -ForegroundColor Yellow
    }
}

# Add to PATH if not already there
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$InstallDir*") {
    Write-Host "Adding to PATH..."
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
}

# Create firewall rule (optional)
$createFirewall = Read-Host "Create firewall rule for port 8080? (y/N)"
if ($createFirewall -eq "y" -or $createFirewall -eq "Y") {
    New-NetFirewallRule -DisplayName "Auth Gate" -Direction Inbound -Protocol TCP -LocalPort 8080 -Action Allow -ErrorAction SilentlyContinue
    Write-Host "Firewall rule created." -ForegroundColor Green
}

Write-Host ""
Write-Host "=== Installation Complete ===" -ForegroundColor Green
Write-Host ""
Write-Host "Binary: $InstallDir\auth-gate.exe"
Write-Host "Config: $InstallDir\config.yaml"
Write-Host ""
Write-Host "Quick start:"
Write-Host "  cd $InstallDir"
Write-Host "  .\auth-gate.exe"
Write-Host ""
Write-Host "Access admin UI at http://localhost:8080/_authgate"
