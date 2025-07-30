
# Existing pyenv-win install logic (if any) goes here...

function Install-PythonVersionFromNexus {
    param (
        [string]$Version
    )

    $arch = "win32"  # Customize if needed
    $nexusRoot = "http://10.88.2.40:8081/repository/python-installers"
    $installer = "python-$Version-$arch.exe"
    $url = "$nexusRoot/$Version/$installer"
    $downloadPath = "$env:TEMP\$installer"

    Write-Host "📥 Downloading Python $Version installer from Nexus..."
    Invoke-WebRequest -Uri $url -OutFile $downloadPath

    Write-Host "🛠 Running installer silently..."
    Start-Process -FilePath $downloadPath -ArgumentList "/quiet InstallAllUsers=1 PrependPath=1 Include_test=0" -Wait

    Write-Host "✅ Python $Version installed from Nexus."
}

# Example usage:
# Install-PythonVersionFromNexus -Version "3.10.11"
