package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pyenv-win/pyenv-win/internal/cache"
	"github.com/pyenv-win/pyenv-win/internal/config"
	"github.com/pyenv-win/pyenv-win/internal/registry"
	"github.com/pyenv-win/pyenv-win/internal/utils"
)

// VersionsCommand handles version-related commands
type VersionsCommand struct {
	cfg *config.PyenvConfig
}

// NewVersionsCommand creates a new versions command
func NewVersionsCommand(cfg *config.PyenvConfig) *VersionsCommand {
	return &VersionsCommand{cfg: cfg}
}

// ListInstalled lists all installed Python versions
func (vc *VersionsCommand) ListInstalled() error {
	currentVersions, _ := vc.getCurrentVersions()
	currentMap := make(map[string]bool)
	for _, v := range currentVersions {
		currentMap[v] = true
	}

	if !utils.DirExists(vc.cfg.VersionsDir) {
		return fmt.Errorf("no versions installed")
	}

	entries, err := utils.ListDir(vc.cfg.VersionsDir)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Println("No Python versions installed")
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		version := entry.Name()
		marker := " "
		if currentMap[version] {
			marker = "*"
		}

		sourcePath := ""
		if currentMap[version] && len(currentVersions) > 0 {
			// Try to find which file set this version
			if versionFile := vc.findVersionFile(version); versionFile != "" {
				sourcePath = fmt.Sprintf(" (set by %s)", versionFile)
			}
		}

		fmt.Printf("%s %s%s\n", marker, version, sourcePath)
	}

	return nil
}

// GetGlobal gets the global Python version
func (vc *VersionsCommand) GetGlobal() ([]string, error) {
	globalVersionFile := filepath.Join(vc.cfg.PyenvHome, "version")
	if !utils.FileExists(globalVersionFile) {
		return nil, fmt.Errorf("no global version configured")
	}

	return vc.readVersionFile(globalVersionFile)
}

// SetGlobal sets the global Python version
func (vc *VersionsCommand) SetGlobal(versions ...string) error {
	// Verify all versions are installed
	for _, version := range versions {
		versionPath := filepath.Join(vc.cfg.VersionsDir, version)
		if !utils.DirExists(versionPath) {
			return fmt.Errorf("version %s is not installed", version)
		}
	}

	globalVersionFile := filepath.Join(vc.cfg.PyenvHome, "version")
	content := strings.Join(versions, "\n") + "\n"

	return utils.WriteFile(globalVersionFile, []byte(content))
}

// GetLocal gets the local Python version from .python-version
func (vc *VersionsCommand) GetLocal(dir string) ([]string, error) {
	if dir == "" {
		dir = "."
	}

	versionFile := filepath.Join(dir, vc.cfg.VersionFile)
	if !utils.FileExists(versionFile) {
		return nil, fmt.Errorf("no local version configured")
	}

	return vc.readVersionFile(versionFile)
}

// SetLocal sets the local Python version in .python-version
func (vc *VersionsCommand) SetLocal(dir string, versions ...string) error {
	if dir == "" {
		dir = "."
	}

	// Verify all versions are installed
	for _, version := range versions {
		versionPath := filepath.Join(vc.cfg.VersionsDir, version)
		if !utils.DirExists(versionPath) {
			return fmt.Errorf("version %s is not installed", version)
		}
	}

	versionFile := filepath.Join(dir, vc.cfg.VersionFile)
	content := strings.Join(versions, "\n") + "\n"

	return utils.WriteFile(versionFile, []byte(content))
}

// UnsetLocal removes the local .python-version file
func (vc *VersionsCommand) UnsetLocal(dir string) error {
	if dir == "" {
		dir = "."
	}

	versionFile := filepath.Join(dir, vc.cfg.VersionFile)
	if !utils.FileExists(versionFile) {
		return nil
	}

	return os.Remove(versionFile)
}

// GetCurrent gets the currently active Python version
func (vc *VersionsCommand) GetCurrent() ([]string, error) {
	return vc.getCurrentVersions()
}

// GetVersionName gets the version name (first current version)
func (vc *VersionsCommand) GetVersionName() (string, error) {
	versions, err := vc.getCurrentVersions()
	if err != nil || len(versions) == 0 {
		return "", fmt.Errorf("no version set")
	}
	return versions[0], nil
}

func (vc *VersionsCommand) getCurrentVersions() ([]string, error) {
	// Check for PYENV_VERSION environment variable
	if envVersion := os.Getenv("PYENV_VERSION"); envVersion != "" {
		return strings.Split(envVersion, ":"), nil
	}

	// Search for .python-version file starting from current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for {
		versionFile := filepath.Join(currentDir, vc.cfg.VersionFile)
		if utils.FileExists(versionFile) {
			return vc.readVersionFile(versionFile)
		}

		// Move to parent directory
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			// Reached root
			break
		}
		currentDir = parent
	}

	// Fall back to global version
	return vc.GetGlobal()
}

func (vc *VersionsCommand) readVersionFile(path string) ([]string, error) {
	data, err := utils.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var versions []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			versions = append(versions, line)
		}
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions in file")
	}

	return versions, nil
}

func (vc *VersionsCommand) findVersionFile(version string) string {
	// Check environment variable
	if envVersion := os.Getenv("PYENV_VERSION"); envVersion != "" {
		if strings.Contains(envVersion, version) {
			return "PYENV_VERSION environment variable"
		}
	}

	// Search for .python-version file
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		versionFile := filepath.Join(currentDir, vc.cfg.VersionFile)
		if utils.FileExists(versionFile) {
			versions, err := vc.readVersionFile(versionFile)
			if err == nil {
				for _, v := range versions {
					if v == version {
						return versionFile
					}
				}
			}
		}

		// Move to parent directory
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break
		}
		currentDir = parent
	}

	// Check global version file
	globalVersionFile := filepath.Join(vc.cfg.PyenvHome, "version")
	if utils.FileExists(globalVersionFile) {
		versions, err := vc.readVersionFile(globalVersionFile)
		if err == nil {
			for _, v := range versions {
				if v == version {
					return globalVersionFile
				}
			}
		}
	}

	return ""
}

// Uninstall removes an installed Python version
func (vc *VersionsCommand) Uninstall(version string, force bool) error {
	versionPath := filepath.Join(vc.cfg.VersionsDir, version)

	if !utils.DirExists(versionPath) {
		return fmt.Errorf("version %s is not installed", version)
	}

	if !force {
		fmt.Printf("pyenv: remove %s? [y/N] ", version)
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("uninstall cancelled")
		}
	}

	utils.PrintInfo("Uninstalling %s...", version)

	// Unregister from Windows registry if it was registered
	if registry.IsRegistered(version) {
		utils.PrintInfo("Unregistering from Windows registry...")
		if err := registry.UnregisterVersion(version); err != nil {
			utils.PrintError("Warning: failed to unregister from registry: %v", err)
			// Continue with uninstall even if registry cleanup fails
		}
	}

	if err := utils.RemoveAll(versionPath); err != nil {
		return fmt.Errorf("failed to remove version: %w", err)
	}

	utils.PrintInfo("Successfully uninstalled %s", version)
	return nil
}

// Latest finds the latest version matching a prefix
func (vc *VersionsCommand) Latest(prefix string, known bool) (string, error) {
	if known {
		// Load from cache
		versionCache, err := cache.LoadVersionsXML(vc.cfg.DBFile)
		if err != nil {
			return "", err
		}

		arch := vc.cfg.GetArchPostfix()
		latest := versionCache.FindLatestVersion(prefix, arch)
		if latest == "" {
			return "", fmt.Errorf("no version found matching: %s", prefix)
		}
		return latest, nil
	}

	// Find from installed versions
	if !utils.DirExists(vc.cfg.VersionsDir) {
		return "", fmt.Errorf("no versions installed")
	}

	entries, err := utils.ListDir(vc.cfg.VersionsDir)
	if err != nil {
		return "", err
	}

	var bestMatch string
	arch := vc.cfg.GetArchPostfix()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		version := entry.Name()

		// Check if version starts with prefix
		if !strings.HasPrefix(version, prefix) {
			continue
		}

		// Full match OR prefix plus '.'
		if version != prefix+arch && !strings.HasPrefix(version, prefix+".") {
			continue
		}

		// Skip dev builds
		if strings.Contains(version, "a") || strings.Contains(version, "b") || strings.Contains(version, "rc") {
			continue
		}

		// Compare with best match
		if bestMatch == "" || cache.CompareVersions(bestMatch, version) {
			bestMatch = version
		}
	}

	if bestMatch == "" {
		return "", fmt.Errorf("no version found matching: %s", prefix)
	}

	return bestMatch, nil
}
