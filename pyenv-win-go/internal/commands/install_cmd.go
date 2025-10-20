package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pyenv-win/pyenv-win/internal/cache"
	"github.com/pyenv-win/pyenv-win/internal/config"
	"github.com/pyenv-win/pyenv-win/internal/install"
	"github.com/pyenv-win/pyenv-win/internal/utils"
)

// InstallCommand handles the install command
type InstallCommand struct {
	cfg       *config.PyenvConfig
	installer *install.Installer
}

// InstallOptions holds options for the install command
type InstallOptions struct {
	List         bool
	All          bool
	Clear        bool
	Force        bool
	SkipExisting bool
	Quiet        bool
	Dev          bool
	Register     bool
	Only32       bool
	Only64       bool
	Offline      bool
	Versions     []string
}

// NewInstallCommand creates a new install command
func NewInstallCommand(cfg *config.PyenvConfig) *InstallCommand {
	return &InstallCommand{
		cfg:       cfg,
		installer: install.NewInstaller(cfg),
	}
}

// Execute runs the install command
func (ic *InstallCommand) Execute(opts *InstallOptions) error {
	// Load version cache
	versionCache, err := cache.LoadVersionsXML(ic.cfg.DBFile)
	if err != nil || len(versionCache.Versions) == 0 {
		utils.PrintError("no definitions in local database")
		fmt.Println()
		fmt.Println("Please update the local database cache with `pyenv update'.")
		return fmt.Errorf("no versions in cache")
	}

	// Handle --list
	if opts.List {
		return ic.listVersions(versionCache)
	}

	// Handle --clear
	if opts.Clear {
		return ic.clearCache()
	}

	// Validate options
	if ic.cfg.Is32Bit() {
		opts.Only32 = false
		opts.Only64 = false
	}

	if opts.Only32 && opts.Only64 {
		return fmt.Errorf("only --32only or --64only may be specified, not both")
	}

	if opts.Register {
		if opts.Only32 {
			return fmt.Errorf("--register not supported for 32 bits")
		}
		if opts.All {
			return fmt.Errorf("--register not supported for all versions")
		}
	}

	// Determine versions to install
	versionsToInstall := opts.Versions

	if opts.All {
		versionsToInstall = ic.getAllVersions(versionCache, opts)
	}

	if len(versionsToInstall) == 0 {
		// Try to get current version
		currentVersions, err := ic.getCurrentVersions()
		if err == nil && len(currentVersions) > 0 {
			versionsToInstall = []string{currentVersions[0]}
		} else {
			ic.showHelp()
			return nil
		}
	}

	// Pre-check all versions exist
	for _, versionCode := range versionsToInstall {
		if versionCache.GetVersion(versionCode) == nil {
			utils.PrintError("definition not found: %s", versionCode)
			fmt.Println()
			fmt.Println("See all available versions with `pyenv install --list`.")
			fmt.Println("Does the list seem out of date? Update it using `pyenv update`.")
			return fmt.Errorf("version not found: %s", versionCode)
		}
	}

	// Install all versions
	for _, versionCode := range versionsToInstall {
		version := versionCache.GetVersion(versionCode)
		if version == nil {
			continue
		}

		params := &install.InstallParams{
			Version:      version,
			InstallPath:  filepath.Join(ic.cfg.VersionsDir, version.Code),
			CacheFile:    filepath.Join(ic.cfg.CacheDir, version.File),
			Quiet:        opts.Quiet,
			Dev:          opts.Dev,
			Offline:      opts.Offline,
			Force:        opts.Force,
			SkipExisting: opts.SkipExisting,
			Register:     opts.Register,
		}

		if err := ic.installer.Install(params); err != nil {
			return fmt.Errorf("failed to install %s: %w", version.Code, err)
		}
	}

	// Rehash
	return ic.rehash()
}

func (ic *InstallCommand) listVersions(versionCache *cache.VersionCache) error {
	for _, version := range versionCache.Versions {
		fmt.Println(version.Code)
	}
	return nil
}

func (ic *InstallCommand) clearCache() error {
	utils.PrintInfo("Clearing install cache...")

	// Remove all files in cache directory
	if !utils.DirExists(ic.cfg.CacheDir) {
		return nil
	}

	entries, err := utils.ListDir(ic.cfg.CacheDir)
	if err != nil {
		return err
	}

	hasError := false
	for _, entry := range entries {
		path := filepath.Join(ic.cfg.CacheDir, entry.Name())
		if err := utils.RemoveAll(path); err != nil {
			utils.PrintError("Error deleting %s: %v", entry.Name(), err)
			hasError = true
		}
	}

	if hasError {
		return fmt.Errorf("some items failed to delete")
	}

	return nil
}

func (ic *InstallCommand) getAllVersions(versionCache *cache.VersionCache, opts *InstallOptions) []string {
	var versions []string

	for _, v := range versionCache.Versions {
		versionCode := v.Code

		// Convert to 32-bit if running on 32-bit platform
		versionCode = cache.Check32Bit(versionCode, ic.cfg.Is32Bit())

		// Check if version exists in cache
		if versionCache.GetVersion(versionCode) == nil {
			continue
		}

		// Filter by architecture
		if opts.Only64 && !v.X64 {
			continue
		}
		if opts.Only32 && v.X64 {
			continue
		}

		versions = append(versions, versionCode)
	}

	return versions
}

func (ic *InstallCommand) getCurrentVersions() ([]string, error) {
	// Read from .python-version file
	versionFile := filepath.Join(".", ic.cfg.VersionFile)
	if utils.FileExists(versionFile) {
		data, err := utils.ReadFile(versionFile)
		if err != nil {
			return nil, err
		}

		lines := strings.Split(string(data), "\n")
		var versions []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				versions = append(versions, line)
			}
		}
		return versions, nil
	}

	// Read from global version file
	globalVersionFile := filepath.Join(ic.cfg.PyenvHome, "version")
	if utils.FileExists(globalVersionFile) {
		data, err := utils.ReadFile(globalVersionFile)
		if err != nil {
			return nil, err
		}

		lines := strings.Split(string(data), "\n")
		var versions []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				versions = append(versions, line)
			}
		}
		return versions, nil
	}

	return nil, fmt.Errorf("no version file found")
}

func (ic *InstallCommand) rehash() error {
	// Rehashing would recreate shims
	// For now, just return nil
	return nil
}

func (ic *InstallCommand) showHelp() {
	fmt.Println("Usage: pyenv install [-s] [-f] <version> [<version> ...] [-r|--register]")
	fmt.Println("       pyenv install [-f] [--32only|--64only] -a|--all")
	fmt.Println("       pyenv install [-f] -c|--clear")
	fmt.Println("       pyenv install -l|--list")
	fmt.Println()
	fmt.Println("  -l/--list              List all available versions")
	fmt.Println("  -a/--all               Installs all known version from the local version DB cache")
	fmt.Println("  -c/--clear             Removes downloaded installers from the cache to free space")
	fmt.Println("  -f/--force             Install even if the version appears to be installed already")
	fmt.Println("  -s/--skip-existing     Skip the installation if the version appears to be installed already")
	fmt.Println("  -r/--register          Register version for py launcher")
	fmt.Println("  -q/--quiet             Install using /quiet. This does not show the UI nor does it prompt for inputs")
	fmt.Println("  --32only               Installs only 32bit Python using -a/--all switch, no effect on 32-bit windows.")
	fmt.Println("  --64only               Installs only 64bit Python using -a/--all switch, no effect on 32-bit windows.")
	fmt.Println("  --dev                  Installs precompiled standard libraries, debug symbols, and debug binaries (only applies to web installer).")
	fmt.Println("  --offline              Download Python installers from Nexus3 server instead of internet")
	fmt.Println("  --help                 Help, list of options allowed on pyenv install")
}
