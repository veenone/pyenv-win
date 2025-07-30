# Nexus3 Repository Setup Guide for pyenv-win Offline Installation

This guide helps Nexus administrators set up a Python installers repository for use with pyenv-win's `--offline` feature.

## 1. Prerequisites

- Nexus Repository Manager 3.x running
- Admin access to Nexus
- Python installer files (EXE format)

## 2. Create Repository

1. Log in to Nexus as an administrator
2. Go to **Settings** (gear icon) → **Repositories**
3. Click **Create repository**
4. Select **raw (hosted)** repository type
5. Configure the repository:
   - **Name**: `python-installers`
   - **Online**: Checked
   - **Blob store**: Select appropriate blob store
   - **Deployment policy**: Allow redeploy (if you need to update installers)

## 3. Upload Python Installers

### Method 1: Web Interface
1. Go to **Browse** → Select your `python-installers` repository
2. Click **Upload component**
3. Create the directory structure and upload files:

```
3.8.10/python-3.8.10-amd64.exe
3.8.10/python-3.8.10-win32.exe
3.9.13/python-3.9.13-amd64.exe
3.9.13/python-3.9.13-win32.exe
3.10.11/python-3.10.11-amd64.exe
3.10.11/python-3.10.11-win32.exe
3.11.7/python-3.11.7-amd64.exe
3.11.7/python-3.11.7-win32.exe
```

### Method 2: REST API
Use curl or PowerShell to upload files:

```bash
# Example with curl
curl -u admin:password \
  --upload-file python-3.10.11-amd64.exe \
  http://your-nexus:8081/repository/python-installers/3.10.11/python-3.10.11-amd64.exe
```

```powershell
# Example with PowerShell
$credentials = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("admin:password"))
$headers = @{
    "Authorization" = "Basic $credentials"
    "Content-Type" = "application/octet-stream"
}

Invoke-RestMethod -Uri "http://your-nexus:8081/repository/python-installers/3.10.11/python-3.10.11-amd64.exe" `
                  -Method Put `
                  -Headers $headers `
                  -InFile "python-3.10.11-amd64.exe"
```

## 4. Download Python Installers

You can download Python installers from:
- Official Python.org: https://www.python.org/downloads/windows/
- Direct links: https://www.python.org/ftp/python/

### Recommended Versions to Upload:
- Python 3.8.10 (amd64 and win32)
- Python 3.9.13 (amd64 and win32)
- Python 3.10.11 (amd64 and win32)
- Python 3.11.7 (amd64 and win32)
- Python 3.12.1 (amd64 and win32)

## 5. Set Permissions

1. Go to **Settings** → **Security** → **Privileges**
2. Create a privilege for the repository:
   - **Name**: `python-installers-read`
   - **Repository**: `python-installers`
   - **Actions**: `read`

3. Go to **Settings** → **Security** → **Roles**
4. Create or modify roles to include the privilege

5. Assign roles to users who need access

## 6. Configure Network Access

Ensure the Nexus server is accessible from client machines:
- Check firewall rules
- Verify DNS resolution
- Test connectivity: `telnet your-nexus-server 8081`

## 7. Test the Setup

1. Set environment variable on client machine:
   ```cmd
   set PYENV_NEXUS_SERVER=http://your-nexus-server:8081/repository/python-installers
   ```

2. Test with pyenv-win:
   ```cmd
   pyenv install --offline 3.10.11
   ```

## 8. Monitoring and Maintenance

### View Repository Statistics
- Go to **Browse** → Select repository → **Summary**
- Monitor download counts and storage usage

### Update Installers
When new Python versions are released:
1. Download new installers from Python.org
2. Upload to appropriate version directories
3. Test with pyenv-win clients

### Backup Strategy
- Regular blob store backups
- Export repository configuration
- Document the directory structure

## 9. Troubleshooting

### Common Issues:

**Connection refused**
- Check Nexus service is running
- Verify firewall settings
- Check network connectivity

**404 Not Found**
- Verify file exists in repository
- Check directory structure matches expected format
- Verify repository name and URL

**Permission denied**
- Check user has read access to repository
- Verify authentication credentials
- Review privilege assignments

### Debugging Commands:

```cmd
# Test direct download
curl -I http://your-nexus:8081/repository/python-installers/3.10.11/python-3.10.11-amd64.exe

# Check repository content
curl http://your-nexus:8081/service/rest/v1/browse/python-installers
```

## 10. Security Considerations

- Use HTTPS in production environments
- Implement proper authentication
- Regular security updates for Nexus
- Monitor access logs
- Consider using signed installers

This setup enables your team to use pyenv-win with `--offline` flag for reliable, fast Python installations without internet dependency.
