# Auth Gate Installer for Windows
# Usage: irm https://raw.githubusercontent.com/pallyoung/auth-gate/main/install.ps1 | iex
# Or:    .\install.ps1 [-InstallDir <path>] [-Version <tag>]

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\auth-gate",
    [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"

$REPO = "pallyoung/auth-gate"

# ── Helpers ──────────────────────────────────────────────────────────────────

function Write-Info  { param([string]$Msg) Write-Host "[INFO]  $Msg" -ForegroundColor Green }
function Write-Warn  { param([string]$Msg) Write-Host "[WARN]  $Msg" -ForegroundColor Yellow }
function Write-Err   { param([string]$Msg) Write-Host "[ERROR] $Msg" -ForegroundColor Red; exit 1 }

function Get-LatestVersion {
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest" -UseBasicParsing
        return $release.tag_name
    } catch {
        Write-Err "Failed to fetch latest version. Check your internet connection."
    }
}

# ── Main ─────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "       Auth Gate Windows Installer      " -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Resolve version
if ($Version -eq "latest") {
    Write-Info "Fetching latest version..."
    $Version = Get-LatestVersion
    if (-not $Version) { Write-Err "Could not determine latest version." }
}
Write-Info "Version: $Version"

# Create temp directory
$tmpDir = Join-Path $env:TEMP "auth-gate-install-$(Get-Random)"
New-Item -ItemType Directory -Force -Path $tmpDir | Out-Null

try {
    # Download release zip
    $assetName = "auth-gate-${Version}-windows-amd64.zip"
    $downloadUrl = "https://github.com/$REPO/releases/download/$Version/$assetName"
    $zipPath = Join-Path $tmpDir $assetName

    Write-Info "Downloading $assetName ..."
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing
    } catch {
        Write-Err "Failed to download $downloadUrl"
    }

    # Extract
    Write-Info "Extracting..."
    Expand-Archive -Path $zipPath -DestinationPath $tmpDir -Force

    # Find extracted folder
    $packageDir = Get-ChildItem -Path $tmpDir -Directory -Filter "auth-gate-*" | Select-Object -First 1
    if (-not $packageDir) {
        # Flat layout fallback
        $packageDir = $tmpDir
    }

    # Install
    Write-Info "Installing to $InstallDir ..."
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -Path "$($packageDir.FullName)\*" -Destination $InstallDir -Recurse -Force

    # Create config from example if missing
    if (-not (Test-Path "$InstallDir\config.yaml")) {
        if (Test-Path "$InstallDir\config.yaml.example") {
            Copy-Item -Path "$InstallDir\config.yaml.example" -Destination "$InstallDir\config.yaml"
            Write-Warn "Created config.yaml from example. Please edit $InstallDir\config.yaml"
        }
    }

    # Add to user PATH
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$InstallDir*") {
        Write-Info "Adding to user PATH..."
        [Environment]::SetEnvironmentVariable("Path", "$currentPath;$InstallDir", "User")
        $env:Path = "$env:Path;$InstallDir"
        Write-Info "PATH updated. Restart your terminal to take effect."
    }

    # Optional: firewall rule
    Write-Host ""
    $createFirewall = Read-Host "Create Windows Firewall rule for port 8080/9000? (y/N)"
    if ($createFirewall -eq "y" -or $createFirewall -eq "Y") {
        try {
            New-NetFirewallRule -DisplayName "Auth Gate (HTTP)" -Direction Inbound -Protocol TCP -LocalPort 8080 -Action Allow -ErrorAction SilentlyContinue
            New-NetFirewallRule -DisplayName "Auth Gate (Admin)" -Direction Inbound -Protocol TCP -LocalPort 9000 -Action Allow -ErrorAction SilentlyContinue
            Write-Info "Firewall rules created for ports 8080 and 9000."
        } catch {
            Write-Warn "Failed to create firewall rule (requires Administrator). Skipping."
        }
    }

    # Optional: register as Windows service via NSSM or sc.exe
    Write-Host ""
    $createService = Read-Host "Register as Windows service? (y/N)"
    if ($createService -eq "y" -or $createService -eq "Y") {
        $serviceName = "auth-gate"
        $exePath = "$InstallDir\auth-gate.exe"

        # Try NSSM first, fall back to sc.exe
        if (Get-Command nssm -ErrorAction SilentlyContinue) {
            Write-Info "Installing service with NSSM..."
            nssm install $serviceName $exePath "serve" "--config" "$InstallDir\config.yaml"
            nssm set $serviceName DisplayName "Auth Gate"
            nssm set $serviceName Description "Auth Gate reverse proxy"
            nssm set $serviceName Start SERVICE_AUTO_START
            nssm start $serviceName
            Write-Info "Service '$serviceName' installed and started."
        } else {
            Write-Warn "NSSM not found. Using sc.exe (basic service registration)..."
            try {
                New-Service -Name $serviceName -BinaryPathName "`"$exePath`" serve --config `"$InstallDir\config.yaml`"" -DisplayName "Auth Gate" -StartupType Automatic -ErrorAction Stop
                Start-Service -Name $serviceName
                Write-Info "Service '$serviceName' registered and started."
            } catch {
                Write-Warn "Failed to register service (requires Administrator). You can run manually: $exePath serve"
            }
        }
    }

} finally {
    # Cleanup temp files
    Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
}

# Done
Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "       Installation Complete!           " -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "  Binary : $InstallDir\auth-gate.exe"
Write-Host "  Config : $InstallDir\config.yaml"
Write-Host ""
Write-Host "  Quick start:"
Write-Host "    cd $InstallDir"
Write-Host "    .\auth-gate.exe serve"
Write-Host ""
Write-Host "  Or with config flag:"
Write-Host "    .\auth-gate.exe serve --config $InstallDir\config.yaml"
Write-Host ""
Write-Host "  Access admin UI at http://localhost:9000"
Write-Host ""
