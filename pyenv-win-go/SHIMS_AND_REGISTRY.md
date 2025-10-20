# Shim Management and Registry Integration

This document provides comprehensive documentation for pyenv-win-go's shim management and Windows registry integration features.

## Table of Contents

- [Shim Management](#shim-management)
  - [What are Shims?](#what-are-shims)
  - [How Shims Work](#how-shims-work)
  - [Shim Commands](#shim-commands)
  - [Implementation Details](#implementation-details)
- [Windows Registry Integration](#windows-registry-integration)
  - [What is Registry Integration?](#what-is-registry-integration)
  - [Benefits](#benefits)
  - [Registry Commands](#registry-commands)
  - [Implementation Details](#registry-implementation-details)
- [Testing](#testing)

---

## Shim Management

### What are Shims?

Shims are lightweight wrapper scripts that intercept calls to Python executables and other Python-related commands. They allow pyenv to automatically route commands to the correct Python version based on the currently active environment.

When you run `python`, `pip`, or any other Python executable, the shim:
1. Determines which Python version is currently active (global, local, or shell)
2. Finds the actual executable for that version
3. Executes the real command with all arguments passed through

### How Shims Work

Each shim is a simple batch file that:
1. Captures its own name (e.g., "python", "pip")
2. Calls `pyenv exec` with that name and all arguments
3. pyenv exec then finds and executes the actual command

Example shim content (`python.bat`):
```batch
@echo off
setlocal
set "PYENV_SHIM=%~n0"
"C:\Users\YourUser\.pyenv\pyenv-win\bin\pyenv" exec "%PYENV_SHIM%" %*
```

### Shim Commands

#### `pyenv rehash`

Recreates all shims for all installed Python versions.

```bash
pyenv rehash
```

**When to use:**
- After installing a new Python version
- After installing new Python packages that provide executables
- If shims become corrupted or out of sync

**Output:**
```
:: [Info] ::  Rehashing shims...
:: [Info] ::  Created 65 shims
```

**What it does:**
1. Clears all existing shims from the shims directory
2. Scans all installed Python versions
3. Finds all executables in:
   - Version root directory (python.exe, pythonw.exe, etc.)
   - Scripts directory (pip.exe, pytest.exe, etc.)
4. Creates a shim batch file for each unique executable

#### `pyenv shims`

Lists all available shims.

```bash
pyenv shims
```

**Example output:**
```
dmypy
docutils
easy_install
idle
pip
pip3
pip3.13
python
python3
python3.13
pytest
virtualenv
wheel
```

**Use cases:**
- Check which commands are shimmed
- Verify shims were created correctly after rehash
- Debug shim-related issues

#### `pyenv which <command>`

Shows the full path to the executable that would be invoked for a command.

```bash
pyenv which python
```

**Example output:**
```
C:\Users\ARaha\.pyenv\pyenv-win\versions\3.13.5\python.exe
```

**Use cases:**
- Verify which version's executable will be used
- Debug PATH issues
- Confirm the active Python version
- Troubleshoot command resolution

**Additional examples:**
```bash
pyenv which pip
# Output: C:\Users\ARaha\.pyenv\pyenv-win\versions\3.13.5\Scripts\pip.exe

pyenv which pytest
# Output: C:\Users\ARaha\.pyenv\pyenv-win\versions\3.13.5\Scripts\pytest.exe
```

#### `pyenv whence <command>`

Lists all Python versions that have a particular command.

```bash
pyenv whence pip
```

**Example output:**
```
3.13.5
3.8.10
3.8.10-win32
3.5.4
3.5.4-win32
```

**Use cases:**
- Find which versions include a specific tool
- Verify package installation across versions
- Debug version-specific command availability

**Additional examples:**
```bash
pyenv whence pytest
# Shows only versions that have pytest installed

pyenv whence python
# Shows all installed versions (python.exe is in all)
```

#### `pyenv exec <command> [args...]`

Executes a command with the current Python environment.

```bash
pyenv exec python --version
pyenv exec pip list
```

**Example output:**
```bash
$ pyenv exec python --version
Python 3.13.5

$ pyenv exec pip --version
pip 25.1.1 from C:\Users\ARaha\.pyenv\pyenv-win\versions\3.13.5\Lib\site-packages\pip (python 3.13)
```

**Use cases:**
- Direct command execution without relying on shims
- Testing command execution
- Scripting that needs explicit pyenv exec calls

### Implementation Details

**Location:** `internal/commands/shims.go`

**Key Functions:**

1. **Rehash()** - Recreates all shims
   - Clears existing shims directory
   - Scans all installed versions
   - Finds executables (.exe, .bat, .cmd files)
   - Creates shim batch files for each executable

2. **Which(command)** - Finds executable path
   - Gets current active version(s)
   - Searches for executable in:
     - Version root directory
     - Scripts subdirectory
   - Returns full path to executable

3. **Whence(command)** - Lists versions with command
   - Scans all installed versions
   - Checks each for the specified command
   - Returns list of version names

4. **Exec(command, args)** - Executes command
   - Uses Which() to find executable
   - Executes with all args passed through
   - Connects stdin/stdout/stderr
   - Preserves exit codes

**Shim Storage:**
- Default location: `%PYENV_HOME%\shims\` or `%USERPROFILE%\.pyenv\pyenv-win\shims\`
- Each shim is a `.bat` file
- Shims directory should be in PATH before Python installations

---

## Windows Registry Integration

### What is Registry Integration?

Windows registry integration allows pyenv-win-go to register installed Python versions with the Windows registry, making them visible to:
- Windows Python launcher (`py.exe`)
- IDEs and development tools
- Python installers and package managers
- System-wide Python detection tools

### Benefits

1. **Python Launcher Integration**
   - Use `py -3.8` to run a specific version
   - Compatible with PEPs for Python launcher behavior

2. **IDE Detection**
   - Visual Studio Code automatically detects registered versions
   - PyCharm and other IDEs can find Python installations
   - No manual interpreter configuration needed

3. **System Integration**
   - Python appears in Windows "Apps & Features"
   - Proper version metadata displayed
   - Standard Windows Python installation behavior

4. **Package Manager Compatibility**
   - pip and other tools can detect multiple Python versions
   - Virtual environment tools work correctly
   - Conda and other Python distributors recognize installations

### Registry Commands

#### Install with Registry (--register flag)

Register a Python version during installation:

```bash
pyenv install --register 3.11.5
```

**What happens:**
1. Python is installed normally
2. Registry keys are created in `HKEY_CURRENT_USER\SOFTWARE\Python\PythonCore\<version>`
3. Version metadata is populated:
   - DisplayName: "Python 3.11 (64-bit)"
   - SysVersion: "3.11"
   - SysArchitecture: "64bit"
   - InstallPath with ExecutablePath
   - PythonPath with Lib and DLLs

**Output:**
```
:: [Info] ::  Registering version 3.11.5 with Windows registry...
:: [Info] ::  Successfully registered version 3.11.5 with Windows registry
```

#### Uninstall with Registry Cleanup

When uninstalling a registered version, registry entries are automatically removed:

```bash
pyenv uninstall 3.11.5
```

**What happens:**
1. Checks if version is registered
2. If registered, removes registry entries
3. Removes the installation directory

**Output:**
```
:: [Info] ::  Uninstalling 3.11.5...
:: [Info] ::  Unregistering from Windows registry...
:: [Info] ::  Successfully uninstalled 3.11.5
```

### Registry Implementation Details

**Location:** `internal/registry/registry.go`

**Key Functions:**

#### RegisterVersion(version, installPath)

Creates registry entries for a Python version.

**Registry Structure Created:**
```
HKEY_CURRENT_USER\SOFTWARE\Python\PythonCore\<version>
  DisplayName = "Python X.Y (64-bit)"
  SupportUrl = "https://github.com/pyenv-win/pyenv-win/issues"
  SysArchitecture = "64bit" or "32bit"
  SysVersion = "X.Y"
  Version = "X.Y.Z"

  InstallPath\
    (Default) = "C:\...\versions\X.Y.Z\"
    ExecutablePath = "C:\...\versions\X.Y.Z\python.exe"
    WindowedExecutablePath = "C:\...\versions\X.Y.Z\pythonw.exe"

  PythonPath\
    (Default) = "C:\...\versions\X.Y.Z\Lib;C:\...\versions\X.Y.Z\DLLs\"
```

**Architecture Detection:**
- Versions ending in `-win32`: 32-bit
- All others: 64-bit (default)
- ARM64 support planned

#### UnregisterVersion(version)

Removes registry entries for a Python version.

**Process:**
1. Deletes InstallPath subkey
2. Deletes PythonPath subkey
3. Deletes main version key
4. Handles missing keys gracefully

#### IsRegistered(version)

Checks if a version is registered.

**Returns:**
- `true` if registry key exists
- `false` otherwise

**Use case:** Check before uninstalling to avoid errors

#### ListRegisteredVersions()

Lists all Python versions registered by pyenv-win-go.

**Returns:** Array of version strings (e.g., `["3.11.5", "3.10.8", "3.9.13"]`)

**Use case:** Audit what's registered, find orphaned entries

#### GetRegisteredPath(version)

Gets the install path for a registered version.

**Returns:** Full path to installation directory

**Use case:** Verify registry accuracy, debug registration issues

---

## Testing

### Shim Management Tests

**Test Shim Creation:**
```bash
# Create shims
pyenv rehash

# Verify shims were created
pyenv shims | head -10

# Check specific shim
pyenv which python
pyenv which pip
```

**Expected Results:**
- 65+ shims created for a typical installation
- `which` commands return correct paths
- Shims in `%PYENV_HOME%\shims\`

**Test Shim Execution:**
```bash
# Execute through pyenv
pyenv exec python --version
pyenv exec pip --version

# Execute through shim (if shims are in PATH)
python --version
pip --version
```

**Expected Results:**
- Commands execute correctly
- Output matches current Python version
- Exit codes preserved

**Test Version Discovery:**
```bash
# Find versions with pip
pyenv whence pip

# Find versions with pytest
pyenv whence pytest
```

**Expected Results:**
- All versions with the command are listed
- Output is accurate and complete

### Registry Integration Tests

**Automated Test Program:**

A comprehensive test program is available in `tests/test_registry.go`:

```bash
cd pyenv-win-go/tests
go build -o test_registry.exe test_registry.go
./test_registry.exe
```

**Test Coverage:**
1. List currently registered versions
2. Check if test version is registered
3. Register a version
4. Verify registration
5. Get registered path
6. List versions again (verify addition)
7. Unregister the version
8. Verify unregistration

**Expected Output:**
```
=== Testing Windows Registry Integration ===

1. Listing currently registered Python versions...
   No versions currently registered

2. Checking if 3.8.10 is registered...
   Result: false

3. Registering 3.8.10...
   Successfully registered!

4. Verifying 3.8.10 is now registered...
   Result: true

5. Getting registered path for 3.8.10...
   Path: C:\Users\ARaha\.pyenv\pyenv-win\versions\3.8.10\

6. Listing registered versions again...
   Found 1 registered versions:
   - 3.8.10 (our test version)

7. Unregistering 3.8.10...
   Successfully unregistered!

8. Verifying 3.8.10 is no longer registered...
   Result: false

=== All registry tests passed! ===
```

**Manual Registry Verification:**

You can manually verify registry entries using Windows Registry Editor:

1. Open Registry Editor (`regedit.exe`)
2. Navigate to: `HKEY_CURRENT_USER\SOFTWARE\Python\PythonCore`
3. Look for your version key (e.g., `3.11.5`)
4. Verify subkeys: `InstallPath`, `PythonPath`
5. Check values match your installation

**Test with Python Launcher:**

If you have `py.exe` installed:

```bash
# List available Python versions
py -0

# Run specific version (if registered)
py -3.11 --version
```

### Integration Testing

**Full Workflow Test:**

```bash
# 1. Install with registry
pyenv install --register 3.10.11

# 2. Verify installation
pyenv versions

# 3. Check registry
# Run test_registry.exe to verify it's registered

# 4. Test shims
pyenv rehash
pyenv which python

# 5. Set as active and test
pyenv global 3.10.11
python --version

# 6. Uninstall (should clean up registry)
pyenv uninstall 3.10.11

# 7. Verify cleanup
# Run test_registry.exe to verify it's unregistered
```

---

## Troubleshooting

### Shims

**Problem:** Shims not working after installation

**Solution:**
```bash
# Recreate shims
pyenv rehash

# Verify shims directory is in PATH
echo %PATH% | findstr shims
```

**Problem:** `pyenv which` returns "executable not found"

**Solution:**
- Check that version is actually installed: `pyenv versions`
- Verify the package provides the executable
- Try rehashing: `pyenv rehash`

**Problem:** Shim executes wrong version

**Solution:**
- Check active version: `pyenv version`
- Verify .python-version files in current directory
- Set explicit version: `pyenv global 3.11.5` or `pyenv local 3.11.5`

### Registry

**Problem:** Version not visible to Python launcher after `--register`

**Solution:**
- Run registry test to verify registration
- Check registry manually in regedit
- Ensure you have permission to write to HKEY_CURRENT_USER

**Problem:** "Access denied" when unregistering

**Solution:**
- This has been fixed in the latest version
- Ensure no other process is accessing the registry key
- Try running as administrator (should not be needed)

**Problem:** Orphaned registry entries after manual deletion

**Solution:**
```bash
# Use the registry test program to clean up
cd pyenv-win-go/tests
go build test_registry.go
./test_registry.exe
# Manually unregister from the code or use regedit
```

---

## Best Practices

### Shims

1. **Always rehash after installing packages**
   ```bash
   pip install pytest
   pyenv rehash
   ```

2. **Keep shims directory first in PATH**
   - Ensures pyenv shims take precedence over system Python

3. **Use `pyenv which` to debug**
   - When commands behave unexpectedly
   - To verify correct version is being used

4. **Don't modify shims manually**
   - Always use `pyenv rehash` to regenerate
   - Manual changes will be overwritten

### Registry

1. **Use `--register` for IDE integration**
   ```bash
   pyenv install --register 3.11.5
   ```

2. **Don't register multiple versions with same major.minor**
   - Windows registry expects one entry per X.Y version
   - Use different versions: 3.11.5, 3.10.8, etc.

3. **Verify registration for critical versions**
   - Use test program or manual registry check
   - Ensures IDE and tool compatibility

4. **Clean up when uninstalling**
   - pyenv handles this automatically
   - Manual verification recommended for critical systems

---

## Performance

### Shim Overhead

Shim execution adds minimal overhead:
- Batch file execution: ~10-20ms
- pyenv exec lookup: ~5-10ms
- Total overhead: ~15-30ms per command

This is negligible for most operations and unnoticeable for long-running commands.

### Registry Performance

Registry operations are fast:
- Registration: ~50-100ms
- Unregistration: ~50-100ms
- Lookup/Check: ~5-10ms

Registry operations are only performed during install/uninstall, not during normal usage.

---

## Future Enhancements

### Planned Features

1. **PowerShell Shims**
   - In addition to batch files
   - Better error handling
   - Enhanced debugging

2. **ARM64 Support**
   - Detect ARM64 architecture
   - Register with correct architecture flag

3. **Registry Export/Import**
   - Backup registry entries
   - Migrate between machines

4. **Bulk Operations**
   - Register all installed versions at once
   - Batch unregister operations

5. **Health Check Command**
   - Verify shims are correct
   - Check registry consistency
   - Report issues and suggest fixes

---

## Credits

- **Shim Design:** Inspired by rbenv and pyenv (Unix)
- **Registry Integration:** Based on official Python installer behavior
- **Implementation:** pyenv-win-go team

---

## License

This documentation is part of pyenv-win-go and is released under the same MIT license.
