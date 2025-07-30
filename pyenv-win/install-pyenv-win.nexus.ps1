
<#
    .SYNOPSIS
    Installs pyenv-win with Nexus offline support

    .DESCRIPTION
    Installs pyenv-win to $HOME\.pyenv with offline Python installer support from Nexus server.
    If pyenv-win is already installed, try to update to the latest version.

    .PARAMETER Uninstall
    Uninstall pyenv-win. Note that this uninstalls any Python versions that were installed with pyenv-win.

    .PARAMETER NexusUrl
    Base URL for Nexus server (default: http://10.88.2.40:8081/repository/python-distribution)

    .PARAMETER Branch
    Git branch to install from (default: feat)

    .PARAMETER LocalPath
    Path to local pyenv-win repository. If specified, copies from local directory instead of downloading from GitHub.

    .INPUTS
    None.

    .OUTPUTS
    None.

    .EXAMPLE
    PS> install-pyenv-win.nexus.ps1

    .EXAMPLE
    PS> install-pyenv-win.nexus.ps1 -NexusUrl "https://nexus.company.com/repository/python-distribution"

    .EXAMPLE
    PS> install-pyenv-win.nexus.ps1 -LocalPath "C:\github\pyenv-win"

    .LINK
    Online version: https://pyenv-win.github.io/pyenv-win/
#>
    
param (
    [Switch] $Uninstall = $False,
    [string] $NexusUrl = "http://10.88.2.40:8081/repository/python-distribution",
    [string] $Branch = "feat",
    [string] $LocalPath = ""
)
    
$PyEnvDir = "${env:USERPROFILE}\.pyenv"
$PyEnvWinDir = "${PyEnvDir}\pyenv-win"
$BinPath = "${PyEnvWinDir}\bin"
$ShimsPath = "${PyEnvWinDir}\shims"
    
Function Remove-PyEnvVars() {
    $PathParts = [System.Environment]::GetEnvironmentVariable('PATH', "User") -Split ";"
    $NewPathParts = $PathParts.Where{ $_ -ne $BinPath }.Where{ $_ -ne $ShimsPath }
    $NewPath = $NewPathParts -Join ";"
    [System.Environment]::SetEnvironmentVariable('PATH', $NewPath, "User")

    [System.Environment]::SetEnvironmentVariable('PYENV', $null, "User")
    [System.Environment]::SetEnvironmentVariable('PYENV_ROOT', $null, "User")
    [System.Environment]::SetEnvironmentVariable('PYENV_HOME', $null, "User")
}

Function Remove-PyEnv() {
    Write-Host "Removing $PyEnvDir..."
    If (Test-Path $PyEnvDir) {
        Remove-Item -Path $PyEnvDir -Recurse
    }
    Write-Host "Removing environment variables..."
    Remove-PyEnvVars
}

Function Get-CurrentVersion() {
    $VersionFilePath = "$PyEnvDir\.version"
    If (Test-Path $VersionFilePath) {
        $CurrentVersion = Get-Content $VersionFilePath
    }
    Else {
        $CurrentVersion = ""
    }

    Return $CurrentVersion
}

Function Get-LatestVersion() {
    $LatestVersionFilePath = "$PyEnvDir\latest.version"
    
    # If LocalPath is specified, try to read version from local repository
    If ($LocalPath -ne "" -and (Test-Path $LocalPath)) {
        $LocalVersionFile = Join-Path $LocalPath ".version"
        If (Test-Path $LocalVersionFile) {
            try {
                $LocalVersion = Get-Content $LocalVersionFile -ErrorAction Stop
                Write-Host "Using version from local repository: $LocalVersion"
                Return $LocalVersion.Trim()
            }
            catch {
                Write-Warning "Could not read version from local repository: $LocalVersionFile"
            }
        }
        else {
            Write-Warning "Version file not found in local repository: $LocalVersionFile"
        }
    }
    
    # Original remote version checking logic
    try {
        # Try to get version from the feat branch instead of master
        (New-Object System.Net.WebClient).DownloadFile("https://raw.githubusercontent.com/pyenv-win/pyenv-win/$Branch/.version", $LatestVersionFilePath)
        $LatestVersion = Get-Content $LatestVersionFilePath
        Remove-Item -Path $LatestVersionFilePath
        Return $LatestVersion
    }
    catch {
        Write-Warning "Could not fetch latest version from $Branch branch, using fallback"
        # Fallback to master if feat branch is not available
        try {
            (New-Object System.Net.WebClient).DownloadFile("https://raw.githubusercontent.com/pyenv-win/pyenv-win/master/.version", $LatestVersionFilePath)
            $LatestVersion = Get-Content $LatestVersionFilePath
            Remove-Item -Path $LatestVersionFilePath
            Return $LatestVersion
        }
        catch {
            Write-Warning "Could not fetch version information. Using 'dev' as fallback."
            Return "dev"
        }
    }
}

Function Configure-NexusSupport() {
    Write-Host "Configuring Nexus offline support..."
    
    # Create pyenv configuration directory if it doesn't exist
    $ConfigDir = "${PyEnvWinDir}\etc"
    If (-not (Test-Path $ConfigDir)) {
        New-Item -ItemType Directory -Path $ConfigDir -Force
    }

    # Create install.ini with Nexus configuration
    $InstallIniPath = "${ConfigDir}\install.ini"
    $InstallIniContent = @"
[install]
python-build-url = $NexusUrl
"@
    
    Set-Content -Path $InstallIniPath -Value $InstallIniContent -Encoding UTF8
    Write-Host "Created install.ini with Nexus URL: $NexusUrl"
    
    # Update mirrors.txt to include Nexus server
    $MirrorsPath = "${PyEnvDir}\mirrors.txt"
    $MirrorsContent = @"
# Python installers from Nexus server (offline)
$NexusUrl

# Official Python releases (online fallback)
https://www.python.org/ftp/python
"@
    
    Set-Content -Path $MirrorsPath -Value $MirrorsContent -Encoding UTF8
    Write-Host "Updated mirrors.txt with Nexus server configuration"
}

Function Main() {
    If ($Uninstall) {
        Remove-PyEnv
        If ($? -eq $True) {
            Write-Host "pyenv-win successfully uninstalled."
        }
        Else {
            Write-Host "Uninstallation failed."
        }
        exit
    }

    $BackupDir = "${env:Temp}/pyenv-win-backup"
    
    $CurrentVersion = Get-CurrentVersion
    If ($CurrentVersion) {
        Write-Host "pyenv-win $CurrentVersion installed."
        $LatestVersion = Get-LatestVersion
        If ($CurrentVersion -eq $LatestVersion -and $LocalPath -eq "") {
            Write-Host "No updates available."
            # Still configure Nexus support even if no update is needed
            Configure-NexusSupport
            Write-Host "Nexus configuration updated."
            exit
        }
        ElseIf ($LocalPath -ne "") {
            Write-Host "Local repository specified, proceeding with installation from local source..."
        }
        Else {
            Write-Host "New version available: $LatestVersion. Updating..."
            
            Write-Host "Backing up existing Python installations..."
            $FoldersToBackup = "install_cache", "versions", "shims"
            ForEach ($Dir in $FoldersToBackup) {
                If (-not (Test-Path $BackupDir)) {
                    New-Item -ItemType Directory -Path $BackupDir
                }
                $SourcePath = "${PyEnvWinDir}/${Dir}"
                If (Test-Path $SourcePath) {
                    Move-Item -Path $SourcePath -Destination $BackupDir
                }
            }
            
            Write-Host "Removing $PyEnvDir..."
            Remove-Item -Path $PyEnvDir -Recurse
        }   
    }

    New-Item -Path $PyEnvDir -ItemType Directory

    # Check if we should use local repository instead of downloading
    If ($LocalPath -ne "" -and (Test-Path $LocalPath)) {
        Write-Host "Using local pyenv-win repository: $LocalPath"
        
        # Validate local repository structure
        $PyEnvWinSource = Join-Path $LocalPath "pyenv-win"
        If (-not (Test-Path $PyEnvWinSource)) {
            Write-Error "Local repository structure invalid. Expected 'pyenv-win' subdirectory in: $LocalPath"
            exit 1
        }
        
        # Copy from local repository
        Write-Host "Copying from local repository..."
        Copy-Item -Path "$LocalPath\*" -Destination $PyEnvDir -Recurse -Force
        
        Write-Host "Successfully copied from local repository."
    }
    Else {
        # Original download logic
        $DownloadPath = "$PyEnvDir\pyenv-win.zip"

        Write-Host "Downloading pyenv-win from $Branch branch..."
        try {
            # Try to download from the specified branch (default: feat)
            (New-Object System.Net.WebClient).DownloadFile("https://github.com/pyenv-win/pyenv-win/archive/$Branch.zip", $DownloadPath)
            $ArchiveFolderName = "pyenv-win-$Branch"
        }
        catch {
            Write-Warning "Could not download from $Branch branch, falling back to master"
            # Fallback to master branch
            (New-Object System.Net.WebClient).DownloadFile("https://github.com/pyenv-win/pyenv-win/archive/master.zip", $DownloadPath)
            $ArchiveFolderName = "pyenv-win-master"
        }

        Write-Host "Extracting pyenv-win..."
        Start-Process -FilePath "powershell.exe" -ArgumentList @(
            "-NoProfile",
            "-Command `"Microsoft.PowerShell.Archive\Expand-Archive -Path \`"$DownloadPath\`" -DestinationPath \`"$PyEnvDir\`"`""
        ) -NoNewWindow -Wait

        Move-Item -Path "$PyEnvDir\$ArchiveFolderName\*" -Destination "$PyEnvDir"
        Remove-Item -Path "$PyEnvDir\$ArchiveFolderName" -Recurse
        Remove-Item -Path $DownloadPath
    }

    # Configure Nexus offline support
    Configure-NexusSupport

    # Update env vars
    [System.Environment]::SetEnvironmentVariable('PYENV', "${PyEnvWinDir}\", "User")
    [System.Environment]::SetEnvironmentVariable('PYENV_ROOT', "${PyEnvWinDir}\", "User")
    [System.Environment]::SetEnvironmentVariable('PYENV_HOME', "${PyEnvWinDir}\", "User")

    $PathParts = [System.Environment]::GetEnvironmentVariable('PATH', "User") -Split ";"

    # Remove existing paths, so we don't add duplicates
    $NewPathParts = $PathParts.Where{ $_ -ne $BinPath }.Where{ $_ -ne $ShimsPath }
    $NewPathParts = ($BinPath, $ShimsPath) + $NewPathParts
    $NewPath = $NewPathParts -Join ";"
    [System.Environment]::SetEnvironmentVariable('PATH', $NewPath, "User")

    If (Test-Path $BackupDir) {
        Write-Host "Restoring Python installations..."
        $BackupItems = Get-ChildItem -Path $BackupDir
        ForEach ($Item in $BackupItems) {
            Move-Item -Path $Item.FullName -Destination $PyEnvWinDir
        }
        Remove-Item -Path $BackupDir -Recurse -Force
    }
    
    If ($? -eq $True) {
        Write-Host "pyenv-win is successfully installed with Nexus offline support."
        If ($LocalPath -ne "") {
            Write-Host "Source: Local repository at $LocalPath"
        } Else {
            Write-Host "Source: GitHub $Branch branch"
        }
        Write-Host "Nexus server configured: $NexusUrl"
        Write-Host ""
        Write-Host "You can now use 'pyenv install --offline <version>' to install Python from Nexus."
        Write-Host "You may need to close and reopen your terminal before using it."
    }
    Else {
        Write-Host "pyenv-win was not installed successfully. If this issue persists, please open a ticket: https://github.com/pyenv-win/pyenv-win/issues."
    }
}

Main
