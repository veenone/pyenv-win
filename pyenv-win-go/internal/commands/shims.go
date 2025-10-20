package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pyenv-win/pyenv-win/internal/config"
	"github.com/pyenv-win/pyenv-win/internal/utils"
)

// ShimsCommand handles shim management
type ShimsCommand struct {
	cfg *config.PyenvConfig
}

// NewShimsCommand creates a new shims command
func NewShimsCommand(cfg *config.PyenvConfig) *ShimsCommand {
	return &ShimsCommand{cfg: cfg}
}

// Rehash recreates all shims for installed Python versions
func (sc *ShimsCommand) Rehash() error {
	utils.PrintInfo("Rehashing shims...")

	// Ensure shims directory exists
	if err := utils.EnsureDir(sc.cfg.ShimsDir); err != nil {
		return fmt.Errorf("failed to create shims directory: %w", err)
	}

	// Clear existing shims
	if err := sc.clearShims(); err != nil {
		return fmt.Errorf("failed to clear existing shims: %w", err)
	}

	// Get all installed versions
	if !utils.DirExists(sc.cfg.VersionsDir) {
		utils.PrintInfo("No Python versions installed yet")
		return nil
	}

	versions, err := utils.ListDir(sc.cfg.VersionsDir)
	if err != nil {
		return fmt.Errorf("failed to list versions: %w", err)
	}

	// Collect all executables from all versions
	executables := make(map[string]bool)

	for _, version := range versions {
		if !version.IsDir() {
			continue
		}

		versionPath := filepath.Join(sc.cfg.VersionsDir, version.Name())

		// Check Scripts directory
		scriptsDir := filepath.Join(versionPath, "Scripts")
		if utils.DirExists(scriptsDir) {
			exes, err := sc.findExecutables(scriptsDir)
			if err == nil {
				for _, exe := range exes {
					executables[exe] = true
				}
			}
		}

		// Check root directory for python executables
		exes, err := sc.findExecutables(versionPath)
		if err == nil {
			for _, exe := range exes {
				if strings.HasPrefix(strings.ToLower(exe), "python") ||
				   strings.HasPrefix(strings.ToLower(exe), "idle") {
					executables[exe] = true
				}
			}
		}
	}

	// Create shims for all executables
	shimCount := 0
	for exe := range executables {
		if err := sc.createShim(exe); err != nil {
			utils.PrintError("Failed to create shim for %s: %v", exe, err)
			continue
		}
		shimCount++
	}

	utils.PrintInfo("Created %d shims", shimCount)
	return nil
}

// ListShims lists all available shims
func (sc *ShimsCommand) ListShims() error {
	if !utils.DirExists(sc.cfg.ShimsDir) {
		return nil
	}

	entries, err := utils.ListDir(sc.cfg.ShimsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".bat") {
			// Print without .bat extension
			name := strings.TrimSuffix(entry.Name(), ".bat")
			fmt.Println(name)
		}
	}

	return nil
}

// Which finds the path to an executable
func (sc *ShimsCommand) Which(command string) (string, error) {
	// Get current version
	versionsCmd := NewVersionsCommand(sc.cfg)
	currentVersions, err := versionsCmd.GetCurrent()
	if err != nil {
		return "", fmt.Errorf("no Python version set")
	}

	// Search in the first (primary) version
	version := currentVersions[0]
	versionPath := filepath.Join(sc.cfg.VersionsDir, version)

	// Try to find the executable
	paths := []string{
		filepath.Join(versionPath, command+".exe"),
		filepath.Join(versionPath, "Scripts", command+".exe"),
		filepath.Join(versionPath, "Scripts", command+".bat"),
		filepath.Join(versionPath, "Scripts", command),
	}

	for _, path := range paths {
		if utils.FileExists(path) {
			return path, nil
		}
	}

	return "", fmt.Errorf("executable not found: %s", command)
}

// Whence lists all versions that contain a command
func (sc *ShimsCommand) Whence(command string) ([]string, error) {
	var versions []string

	if !utils.DirExists(sc.cfg.VersionsDir) {
		return versions, nil
	}

	entries, err := utils.ListDir(sc.cfg.VersionsDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		versionPath := filepath.Join(sc.cfg.VersionsDir, entry.Name())

		// Check if this version has the command
		paths := []string{
			filepath.Join(versionPath, command+".exe"),
			filepath.Join(versionPath, "Scripts", command+".exe"),
			filepath.Join(versionPath, "Scripts", command+".bat"),
		}

		for _, path := range paths {
			if utils.FileExists(path) {
				versions = append(versions, entry.Name())
				break
			}
		}
	}

	return versions, nil
}

func (sc *ShimsCommand) clearShims() error {
	if !utils.DirExists(sc.cfg.ShimsDir) {
		return nil
	}

	entries, err := utils.ListDir(sc.cfg.ShimsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			path := filepath.Join(sc.cfg.ShimsDir, entry.Name())
			if err := os.Remove(path); err != nil {
				utils.PrintError("Failed to remove %s: %v", entry.Name(), err)
			}
		}
	}

	return nil
}

func (sc *ShimsCommand) findExecutables(dir string) ([]string, error) {
	var executables []string

	entries, err := utils.ListDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))

		// Include .exe, .bat, .cmd files
		if ext == ".exe" || ext == ".bat" || ext == ".cmd" {
			// Get name without extension
			baseName := strings.TrimSuffix(name, ext)
			executables = append(executables, baseName)
		}
	}

	return executables, nil
}

func (sc *ShimsCommand) createShim(exeName string) error {
	shimPath := filepath.Join(sc.cfg.ShimsDir, exeName+".bat")

	// Create shim batch file that delegates to pyenv exec
	shimContent := fmt.Sprintf(`@echo off
setlocal
set "PYENV_SHIM=%%~n0"
"%s\bin\pyenv" exec "%%PYENV_SHIM%%" %%*
`, sc.cfg.PyenvHome)

	return utils.WriteFile(shimPath, []byte(shimContent))
}

// Exec executes a command with the current Python environment
func (sc *ShimsCommand) Exec(command string, args []string) error {
	// Find the executable path
	execPath, err := sc.Which(command)
	if err != nil {
		return err
	}

	// Execute the command and pass through stdin/stdout/stderr
	cmd := exec.Command(execPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// If the command fails, exit with the same code
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}
