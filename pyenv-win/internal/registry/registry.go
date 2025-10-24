package registry

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// RegisterVersion registers a Python version in Windows registry for py launcher
func RegisterVersion(version, installPath string) error {
	// Parse version to get major.minor
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid version format: %s", version)
	}

	major := parts[0]
	minor := parts[1]

	// Remove any non-numeric suffix from minor version
	for i, c := range minor {
		if c < '0' || c > '9' {
			minor = minor[:i]
			break
		}
	}

	sysVersion := major + "." + minor

	// Determine architecture
	bitDepth := "64"
	versionAttribute := version
	if strings.HasSuffix(version, "-win32") {
		bitDepth = "32"
		versionAttribute = strings.TrimSuffix(version, "-win32")
	}

	// Open or create registry key
	keyPath := `SOFTWARE\Python\PythonCore\` + version
	key, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to create registry key: %w", err)
	}
	defer key.Close()

	// Set display name
	displayName := fmt.Sprintf("Python %s (%s-bit)", sysVersion, bitDepth)
	if err := key.SetStringValue("DisplayName", displayName); err != nil {
		return fmt.Errorf("failed to set DisplayName: %w", err)
	}

	// Set support URL
	if err := key.SetStringValue("SupportUrl", "https://github.com/pyenv-win/pyenv-win/issues"); err != nil {
		return fmt.Errorf("failed to set SupportUrl: %w", err)
	}

	// Set system architecture
	if err := key.SetStringValue("SysArchitecture", bitDepth+"bit"); err != nil {
		return fmt.Errorf("failed to set SysArchitecture: %w", err)
	}

	// Set system version
	if err := key.SetStringValue("SysVersion", sysVersion); err != nil {
		return fmt.Errorf("failed to set SysVersion: %w", err)
	}

	// Set version
	if err := key.SetStringValue("Version", versionAttribute); err != nil {
		return fmt.Errorf("failed to set Version: %w", err)
	}

	// Create InstallPath subkey
	installPathKey, _, err := registry.CreateKey(key, "InstallPath", registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to create InstallPath key: %w", err)
	}
	defer installPathKey.Close()

	// Set install path (must end with backslash)
	installPathStr := installPath
	if !strings.HasSuffix(installPathStr, "\\") {
		installPathStr += "\\"
	}
	if err := installPathKey.SetStringValue("", installPathStr); err != nil {
		return fmt.Errorf("failed to set InstallPath: %w", err)
	}

	// Set executable path
	if err := installPathKey.SetStringValue("ExecutablePath", filepath.Join(installPath, "python.exe")); err != nil {
		return fmt.Errorf("failed to set ExecutablePath: %w", err)
	}

	// Set windowed executable path
	if err := installPathKey.SetStringValue("WindowedExecutablePath", filepath.Join(installPath, "pythonw.exe")); err != nil {
		return fmt.Errorf("failed to set WindowedExecutablePath: %w", err)
	}

	// Create PythonPath subkey
	pythonPathKey, _, err := registry.CreateKey(key, "PythonPath", registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to create PythonPath key: %w", err)
	}
	defer pythonPathKey.Close()

	// Set Python path
	pythonPath := filepath.Join(installPath, "Lib") + ";" + filepath.Join(installPath, "DLLs") + "\\"
	if err := pythonPathKey.SetStringValue("", pythonPath); err != nil {
		return fmt.Errorf("failed to set PythonPath: %w", err)
	}

	return nil
}

// UnregisterVersion removes a Python version from Windows registry
func UnregisterVersion(version string) error {
	keyPath := `SOFTWARE\Python\PythonCore\` + version

	// Open the parent key
	parentPath := `SOFTWARE\Python\PythonCore`
	parentKey, err := registry.OpenKey(registry.CURRENT_USER, parentPath, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			// Parent key doesn't exist, nothing to delete
			return nil
		}
		return fmt.Errorf("failed to open parent key: %w", err)
	}
	defer parentKey.Close()

	// Delete subkeys first (InstallPath, PythonPath)
	subkeys := []string{"InstallPath", "PythonPath"}
	for _, subkey := range subkeys {
		subkeyPath := keyPath + `\` + subkey
		registry.DeleteKey(registry.CURRENT_USER, subkeyPath)
		// Ignore errors - subkeys might not exist
	}

	// Now delete the main key
	err = registry.DeleteKey(registry.CURRENT_USER, keyPath)
	if err != nil {
		// Key might not exist, which is fine
		if err != registry.ErrNotExist {
			return fmt.Errorf("failed to delete registry key: %w", err)
		}
	}

	return nil
}

// IsRegistered checks if a version is registered in the registry
func IsRegistered(version string) bool {
	keyPath := `SOFTWARE\Python\PythonCore\` + version
	key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	key.Close()
	return true
}

// ListRegisteredVersions lists all Python versions registered in the registry
func ListRegisteredVersions() ([]string, error) {
	keyPath := `SOFTWARE\Python\PythonCore`
	key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		if err == registry.ErrNotExist {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	subkeys, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read subkeys: %w", err)
	}

	return subkeys, nil
}

// GetRegisteredPath gets the install path for a registered version
func GetRegisteredPath(version string) (string, error) {
	keyPath := `SOFTWARE\Python\PythonCore\` + version + `\InstallPath`
	key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	path, _, err := key.GetStringValue("")
	if err != nil {
		return "", fmt.Errorf("failed to read InstallPath: %w", err)
	}

	return path, nil
}
