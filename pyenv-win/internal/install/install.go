package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pyenv-win/pyenv-win/internal/cache"
	"github.com/pyenv-win/pyenv-win/internal/config"
	"github.com/pyenv-win/pyenv-win/internal/download"
	"github.com/pyenv-win/pyenv-win/internal/registry"
	"github.com/pyenv-win/pyenv-win/internal/utils"
)

// Installer handles Python version installations
type Installer struct {
	cfg    *config.PyenvConfig
	client *download.Client
}

// NewInstaller creates a new installer
func NewInstaller(cfg *config.PyenvConfig) *Installer {
	return &Installer{
		cfg:    cfg,
		client: download.NewClient(cfg.HTTPProxy, cfg.HTTPSProxy),
	}
}

// InstallParams holds installation parameters
type InstallParams struct {
	Version      *cache.Version
	InstallPath  string
	CacheFile    string
	Quiet        bool
	Dev          bool
	Offline      bool
	Force        bool
	SkipExisting bool
	Register     bool
}

// Install installs a Python version
func (i *Installer) Install(params *InstallParams) error {
	// Check if already installed
	if utils.DirExists(params.InstallPath) {
		if params.SkipExisting {
			utils.PrintInfo("Version %s already installed, skipping", params.Version.Code)
			return nil
		}
		if !params.Force {
			return fmt.Errorf("version %s already installed (use -f to reinstall)", params.Version.Code)
		}
		// Remove existing installation
		utils.PrintInfo("Removing existing installation")
		if err := utils.RemoveAll(params.InstallPath); err != nil {
			return fmt.Errorf("failed to remove existing installation: %w", err)
		}
	}

	// Ensure cache directory exists
	if err := utils.EnsureDir(i.cfg.CacheDir); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Ensure versions directory exists
	versionsParent := filepath.Dir(params.InstallPath)
	if err := utils.EnsureDir(versionsParent); err != nil {
		return fmt.Errorf("failed to create versions directory: %w", err)
	}

	// Download if needed
	if !utils.FileExists(params.CacheFile) {
		if err := i.download(params); err != nil {
			return err
		}
	}

	// Install
	if err := i.extract(params); err != nil {
		return err
	}

	// Register with Windows if requested
	if params.Register {
		if err := i.register(params); err != nil {
			return fmt.Errorf("failed to register version: %w", err)
		}
	}

	utils.PrintInfo("completed! %s", params.Version.Code)
	return nil
}

func (i *Installer) download(params *InstallParams) error {
	url := params.Version.URL

	// Use Nexus server if offline mode
	if params.Offline {
		if i.cfg.NexusServer == "" {
			return fmt.Errorf("PYENV_NEXUS_SERVER environment variable is not set")
		}
		url = i.buildNexusURL(params.Version.Code, params.Version.File)
		utils.PrintDownloading("%s (offline mode)...", params.Version.Code)
		utils.PrintDownloading("From Nexus: %s", url)
	} else {
		utils.PrintDownloading("%s ...", params.Version.Code)
		utils.PrintDownloading("From %s", url)
	}

	utils.PrintDownloading("To   %s", params.CacheFile)

	if err := i.client.DownloadFile(url, params.CacheFile); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	return nil
}

func (i *Installer) buildNexusURL(version, filename string) string {
	nexusServer := i.cfg.NexusServer

	// Remove architecture suffix from version
	versionNumber := version
	versionNumber = strings.TrimSuffix(versionNumber, "-win32")
	versionNumber = strings.TrimSuffix(versionNumber, "-arm")

	// Build URL: {nexusServer}/{version}/{filename}
	if strings.HasSuffix(nexusServer, "/") {
		return nexusServer + versionNumber + "/" + filename
	}
	return nexusServer + "/" + versionNumber + "/" + filename
}

func (i *Installer) extract(params *InstallParams) error {
	utils.PrintInstalling("%s ...", params.Version.Code)

	if params.Version.MSI {
		return i.extractMSI(params)
	}

	ext := filepath.Ext(params.CacheFile)
	if ext == ".zip" {
		return i.extractZip(params)
	}

	// Handle .exe installers
	return i.extractEXE(params)
}

func (i *Installer) extractMSI(params *InstallParams) error {
	// Use msiexec for admin install
	cmd := exec.Command("msiexec",
		"/quiet",
		"/a", params.CacheFile,
		"TargetDir="+params.InstallPath,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("msiexec failed: %w", err)
	}

	// Remove duplicate .msi files from install path
	msiFiles, _ := utils.FindFilesWithExt(params.InstallPath, ".msi")
	for _, msi := range msiFiles {
		os.Remove(msi)
	}

	// Install pip if ensurepip exists
	pythonExe := filepath.Join(params.InstallPath, "python.exe")
	ensurepipDir := filepath.Join(params.InstallPath, "Lib", "ensurepip")

	if utils.DirExists(ensurepipDir) {
		pipCmd := exec.Command(pythonExe, "-E", "-s", "-m", "ensurepip", "-U", "--default-pip")
		if err := pipCmd.Run(); err != nil {
			return fmt.Errorf("failed to install pip: %w", err)
		}
	}

	return i.createVersionAliases(params)
}

func (i *Installer) extractZip(params *InstallParams) error {
	// Extract zip using PowerShell Expand-Archive or similar
	// For Windows, we can use built-in functionality

	// Create parent directory
	if err := utils.EnsureDir(filepath.Dir(params.InstallPath)); err != nil {
		return err
	}

	// Use PowerShell to extract
	psCmd := fmt.Sprintf(`Expand-Archive -Path '%s' -DestinationPath '%s' -Force`,
		params.CacheFile, filepath.Dir(params.InstallPath))

	cmd := exec.Command("powershell", "-Command", psCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract zip: %w", err)
	}

	// If there's a zipRootDir, rename it
	if params.Version.ZipRootDir != "" {
		srcDir := filepath.Join(filepath.Dir(params.InstallPath), params.Version.ZipRootDir)
		if utils.DirExists(srcDir) && srcDir != params.InstallPath {
			if err := os.Rename(srcDir, params.InstallPath); err != nil {
				return fmt.Errorf("failed to move extracted directory: %w", err)
			}
		}
	}

	return nil
}

func (i *Installer) extractEXE(params *InstallParams) error {
	// For .exe installers, we need to use dark.exe (WiX) for extraction
	// or handle web installers differently

	if params.Version.WebInstall {
		return i.extractWebInstaller(params)
	}

	return i.extractEmbeddedInstaller(params)
}

func (i *Installer) extractWebInstaller(params *InstallParams) error {
	cachePath := filepath.Join(i.cfg.CacheDir, params.Version.Code+"-webinstall")

	if !utils.DirExists(cachePath) {
		// Run web installer with /quiet /layout
		cmd := exec.Command(params.CacheFile, "/quiet", "/layout", cachePath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to extract web installer: %w", err)
		}
	}

	return i.installFromCache(cachePath, params)
}

func (i *Installer) extractEmbeddedInstaller(params *InstallParams) error {
	cachePath := filepath.Join(i.cfg.CacheDir, params.Version.Code)

	if !utils.DirExists(cachePath) {
		darkExe := filepath.Join(i.cfg.WixDir, "dark.exe")

		// Extract using dark.exe
		cmd := exec.Command(darkExe, "-x", cachePath, params.CacheFile)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to extract embedded installer: %w", err)
		}

		// Move MSI files from AttachedContainer
		containerPath := filepath.Join(cachePath, "AttachedContainer")
		if utils.DirExists(containerPath) {
			msiFiles, err := utils.FindFilesWithExt(containerPath, ".msi")
			if err == nil {
				for _, msi := range msiFiles {
					dest := filepath.Join(cachePath, filepath.Base(msi))
					utils.MoveFile(msi, dest)
				}
			}
		}
	}

	return i.installFromCache(cachePath, params)
}

func (i *Installer) installFromCache(cachePath string, params *InstallParams) error {
	// Clean unused install files
	files, err := utils.ListDir(cachePath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			// Remove subdirectories
			os.RemoveAll(filepath.Join(cachePath, file.Name()))
			continue
		}

		baseName := strings.ToLower(strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())))
		ext := strings.ToLower(filepath.Ext(file.Name()))

		// Remove non-MSI files and specific MSI files
		if ext != ".msi" || baseName == "appendpath" || baseName == "launcher" ||
		   baseName == "path" || baseName == "pip" {
			os.Remove(filepath.Join(cachePath, file.Name()))
		}
	}

	// Install remaining MSI files
	msiFiles, err := utils.FindFilesWithExt(cachePath, ".msi")
	if err != nil {
		return err
	}

	for _, msi := range msiFiles {
		cmd := exec.Command("msiexec", "/quiet", "/a", msi,
			"TargetDir="+params.InstallPath)
		if err := cmd.Run(); err != nil {
			baseName := filepath.Base(msi)
			return fmt.Errorf("failed to install %s component MSI: %w", baseName, err)
		}

		// Delete duplicate MSI file from install path
		destMSI := filepath.Join(params.InstallPath, filepath.Base(msi))
		if utils.FileExists(destMSI) {
			os.Remove(destMSI)
		}
	}

	// Install pip if ensurepip exists
	pythonExe := filepath.Join(params.InstallPath, "python.exe")
	ensurepipDir := filepath.Join(params.InstallPath, "Lib", "ensurepip")

	if utils.DirExists(ensurepipDir) {
		pipCmd := exec.Command(pythonExe, "-E", "-s", "-m", "ensurepip", "-U", "--default-pip")
		if err := pipCmd.Run(); err != nil {
			return fmt.Errorf("failed to install pip: %w", err)
		}
	}

	return i.createVersionAliases(params)
}

func (i *Installer) createVersionAliases(params *InstallParams) error {
	// Create pythonX, pythonXY, pythonX.Y executables
	version := params.Version.Code
	pythonExe := filepath.Join(params.InstallPath, "python.exe")
	pythonwExe := filepath.Join(params.InstallPath, "pythonw.exe")

	if !utils.FileExists(pythonExe) {
		return fmt.Errorf("python.exe not found in %s", params.InstallPath)
	}

	// Parse version to get major.minor
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return nil // Can't create aliases without major.minor
	}

	major := parts[0]
	minor := parts[1]
	// Remove any non-numeric suffix from minor
	for i, c := range minor {
		if c < '0' || c > '9' {
			minor = minor[:i]
			break
		}
	}

	majorMinor := major + minor
	majorDotMinor := major + "." + minor

	// Create python aliases
	aliases := []string{
		filepath.Join(params.InstallPath, "python"+major+".exe"),
		filepath.Join(params.InstallPath, "python"+majorMinor+".exe"),
		filepath.Join(params.InstallPath, "python"+majorDotMinor+".exe"),
	}

	for _, alias := range aliases {
		utils.CopyFile(pythonExe, alias)
	}

	// Create pythonw aliases
	if utils.FileExists(pythonwExe) {
		aliases = []string{
			filepath.Join(params.InstallPath, "pythonw"+major+".exe"),
			filepath.Join(params.InstallPath, "pythonw"+majorMinor+".exe"),
			filepath.Join(params.InstallPath, "pythonw"+majorDotMinor+".exe"),
		}

		for _, alias := range aliases {
			utils.CopyFile(pythonwExe, alias)
		}
	}

	// Create venv launcher aliases if they exist
	venvLauncherExe := filepath.Join(params.InstallPath, "Lib", "venv", "scripts", "nt", "python.exe")
	if utils.FileExists(venvLauncherExe) {
		venvDir := filepath.Dir(venvLauncherExe)
		aliases = []string{
			filepath.Join(venvDir, "python"+major+".exe"),
			filepath.Join(venvDir, "python"+majorMinor+".exe"),
			filepath.Join(venvDir, "python"+majorDotMinor+".exe"),
			filepath.Join(venvDir, "pythonw"+major+".exe"),
			filepath.Join(venvDir, "pythonw"+majorMinor+".exe"),
			filepath.Join(venvDir, "pythonw"+majorDotMinor+".exe"),
		}

		for _, alias := range aliases {
			utils.CopyFile(venvLauncherExe, alias)
		}
	}

	return nil
}

func (i *Installer) register(params *InstallParams) error {
	utils.PrintInfo("Registering version %s with Windows registry...", params.Version.Code)

	if err := registry.RegisterVersion(params.Version.Code, params.InstallPath); err != nil {
		return fmt.Errorf("failed to register with Windows registry: %w", err)
	}

	utils.PrintInfo("Successfully registered version %s with Windows registry", params.Version.Code)
	return nil
}

// Clear removes an installation and its cache file
func (i *Installer) Clear(params *InstallParams) error {
	if utils.DirExists(params.InstallPath) {
		if err := utils.RemoveAll(params.InstallPath); err != nil {
			return err
		}
	}

	if utils.FileExists(params.CacheFile) {
		if err := os.Remove(params.CacheFile); err != nil {
			return err
		}
	}

	return nil
}
