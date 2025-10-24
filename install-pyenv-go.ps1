# Install pyenv-win-go (Go-based implementation)
# This script installs the Go binary to replace the VBScript-based pyenv

param(
    [switch]$Mirror = $false,
    [switch]$Force = $false
)

$ErrorActionPreference = "Stop"

# Detect PYENV_HOME
$PyenvHome = $env:PYENV
if (-not $PyenvHome) {
    $PyenvHome = $env:PYENV_HOME
}
if (-not $PyenvHome) {
    $PyenvHome = "$env:USERPROFILE\.pyenv\pyenv-win"
}

# Create pyenv home directory if it doesn't exist
if (-not (Test-Path $PyenvHome)) {
    Write-Host "Creating pyenv directory structure at: $PyenvHome" -ForegroundColor Yellow
    New-Item -ItemType Directory -Path $PyenvHome -Force | Out-Null
    New-Item -ItemType Directory -Path "$PyenvHome\bin" -Force | Out-Null
    New-Item -ItemType Directory -Path "$PyenvHome\shims" -Force | Out-Null
    New-Item -ItemType Directory -Path "$PyenvHome\versions" -Force | Out-Null
    New-Item -ItemType Directory -Path "$PyenvHome\install_cache" -Force | Out-Null
    Write-Host ""
}

Write-Host "Installing pyenv-win-go to: $PyenvHome" -ForegroundColor Cyan
Write-Host ""

# Determine which binary to install
$SourceBinary = ".\build\pyenv.exe"
if ($Mirror) {
    $SourceBinary = ".\build\pyenv-mirror.exe"
    Write-Host "Installing MIRROR version (with Nexus support)" -ForegroundColor Yellow
} else {
    Write-Host "Installing STANDARD version" -ForegroundColor Green
}

if (-not (Test-Path $SourceBinary)) {
    Write-Host "ERROR: Binary not found: $SourceBinary" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please build pyenv first:"
    Write-Host "  make build          # For standard version"
    Write-Host "  make build-mirror   # For mirror version"
    exit 1
}

# Backup old files
$BinDir = "$PyenvHome\bin"
$BackupDir = "$PyenvHome\bin\backup-$(Get-Date -Format 'yyyyMMdd-HHmmss')"

if (-not $Force) {
    Write-Host "Creating backup of old pyenv files..." -ForegroundColor Yellow
    New-Item -ItemType Directory -Path $BackupDir -Force | Out-Null

    if (Test-Path "$BinDir\pyenv.bat") {
        Copy-Item "$BinDir\pyenv.bat" "$BackupDir\" -Force
        Write-Host "  Backed up: pyenv.bat"
    }
    if (Test-Path "$BinDir\pyenv") {
        Copy-Item "$BinDir\pyenv" "$BackupDir\" -Force
        Write-Host "  Backed up: pyenv (shell script)"
    }
    if (Test-Path "$BinDir\pyenv.ps1") {
        Copy-Item "$BinDir\pyenv.ps1" "$BackupDir\" -Force
        Write-Host "  Backed up: pyenv.ps1"
    }
    Write-Host ""
}

# Remove old libexec directory (VBScript files)
if (Test-Path "$PyenvHome\libexec") {
    Write-Host "Removing old VBScript files from libexec..." -ForegroundColor Yellow
    Remove-Item "$PyenvHome\libexec" -Recurse -Force
    Write-Host "  Removed: libexec directory"
    Write-Host ""
}

# Install Go binary
Write-Host "Installing Go binary..." -ForegroundColor Green
$DestBinary = "$BinDir\pyenv.exe"
Copy-Item $SourceBinary $DestBinary -Force
Write-Host "  Installed: $DestBinary"
Write-Host ""

# Update wrapper scripts to call the Go binary
Write-Host "Updating wrapper scripts..." -ForegroundColor Green

# Update pyenv.bat
$PyenvBat = @"
@echo off
setlocal
"%~dp0pyenv.exe" %*
"@
Set-Content -Path "$BinDir\pyenv.bat" -Value $PyenvBat -Force
Write-Host "  Updated: pyenv.bat"

# Update pyenv shell script (for Git Bash / Cygwin)
$PyenvSh = @"
#!/bin/sh
SCRIPT_DIR="`$(dirname "`$0")"
"`$SCRIPT_DIR/pyenv.exe" "`$@"
"@
Set-Content -Path "$BinDir\pyenv" -Value $PyenvSh -Force -NoNewline
Write-Host "  Updated: pyenv (shell script)"

# Update pyenv.ps1
$PyenvPs1 = @"
# PowerShell wrapper for pyenv-win-go
`$PyenvExe = Join-Path `$PSScriptRoot "pyenv.exe"
& `$PyenvExe @args
"@
Set-Content -Path "$BinDir\pyenv.ps1" -Value $PyenvPs1 -Force
Write-Host "  Updated: pyenv.ps1"
Write-Host ""

# Check for WiX tools (needed for .exe installer extraction)
$WixDir = "$BinDir\WiX"
if (-not (Test-Path "$WixDir\dark.exe")) {
    Write-Host "WiX tools not found. Downloading..." -ForegroundColor Yellow
    Write-Host "NOTE: WiX tools are required to install Python from .exe installers" -ForegroundColor Yellow
    Write-Host ""

    # Create WiX directory
    New-Item -ItemType Directory -Path $WixDir -Force | Out-Null

    # Download WiX tools from pyenv-win repository
    $WixUrl = "https://github.com/pyenv-win/pyenv-win/raw/master/pyenv-win/bin/WiX/dark.exe"
    $DarkExePath = "$WixDir\dark.exe"

    try {
        Write-Host "  Downloading dark.exe..."
        Invoke-WebRequest -Uri $WixUrl -OutFile $DarkExePath -UseBasicParsing
        Write-Host "  Downloaded: dark.exe" -ForegroundColor Green
        Write-Host ""
    } catch {
        Write-Host "  WARNING: Failed to download WiX tools" -ForegroundColor Yellow
        Write-Host "  You may need to install WiX manually to use .exe installers" -ForegroundColor Yellow
        Write-Host "  MSI and ZIP installers will still work" -ForegroundColor Yellow
        Write-Host ""
    }
}

# Check for .versions_cache.xml
$CacheFile = "$PyenvHome\.versions_cache.xml"
if (-not (Test-Path $CacheFile)) {
    Write-Host "Downloading version cache..." -ForegroundColor Yellow
    $CacheUrl = "https://github.com/pyenv-win/pyenv-win/raw/master/pyenv-win/.versions_cache.xml"

    try {
        Invoke-WebRequest -Uri $CacheUrl -OutFile $CacheFile -UseBasicParsing
        Write-Host "  Downloaded: .versions_cache.xml" -ForegroundColor Green
        Write-Host ""
    } catch {
        Write-Host "  WARNING: Failed to download version cache" -ForegroundColor Yellow
        Write-Host "  Run 'pyenv update' to download the version list" -ForegroundColor Yellow
        Write-Host ""
    }
}

# Verify installation
Write-Host "Verifying installation..." -ForegroundColor Cyan
$Version = & "$DestBinary" --version
if ($LASTEXITCODE -eq 0) {
    Write-Host "  $Version" -ForegroundColor Green
    Write-Host ""
    Write-Host "Installation successful!" -ForegroundColor Green
    Write-Host ""

    # Check if PATH is configured
    $PathEnv = [Environment]::GetEnvironmentVariable("Path", "User")
    $PyenvInPath = $PathEnv -like "*$PyenvHome\bin*" -or $PathEnv -like "*$PyenvHome\shims*"

    if (-not $PyenvInPath) {
        Write-Host "IMPORTANT: Add pyenv to your PATH" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Add these paths to your System Environment Variables (User PATH):" -ForegroundColor Cyan
        Write-Host "  1. $PyenvHome\bin" -ForegroundColor White
        Write-Host "  2. $PyenvHome\shims" -ForegroundColor White
        Write-Host ""
        Write-Host "Quick setup (run as administrator):" -ForegroundColor Cyan
        Write-Host "  [Environment]::SetEnvironmentVariable('PYENV', '$PyenvHome', 'User')" -ForegroundColor White
        Write-Host "  [Environment]::SetEnvironmentVariable('PYENV_HOME', '$PyenvHome', 'User')" -ForegroundColor White
        Write-Host "  [Environment]::SetEnvironmentVariable('PYENV_ROOT', '$PyenvHome', 'User')" -ForegroundColor White
        Write-Host "  `$Path = [Environment]::GetEnvironmentVariable('Path', 'User')" -ForegroundColor White
        Write-Host "  `$NewPath = `"$PyenvHome\bin;$PyenvHome\shims;`$Path`"" -ForegroundColor White
        Write-Host "  [Environment]::SetEnvironmentVariable('Path', `$NewPath, 'User')" -ForegroundColor White
        Write-Host ""
        Write-Host "Then restart your terminal and run:" -ForegroundColor Cyan
        Write-Host "  pyenv --version" -ForegroundColor White
        Write-Host ""
    }

    Write-Host "You can now use pyenv commands:" -ForegroundColor Cyan
    Write-Host "  pyenv install --list         # List available Python versions"
    Write-Host "  pyenv install 3.12.0         # Install Python 3.12.0"
    Write-Host "  pyenv global 3.12.0          # Set global Python version"
    Write-Host "  pyenv versions               # List installed versions"
    Write-Host ""
    if ($Mirror) {
        Write-Host "Mirror commands are available:" -ForegroundColor Cyan
        Write-Host "  pyenv mirror init            # Create mirror config"
        Write-Host "  pyenv mirror all             # Mirror all versions to Nexus"
        Write-Host ""
    }
} else {
    Write-Host "ERROR: Installation verification failed" -ForegroundColor Red
    exit 1
}