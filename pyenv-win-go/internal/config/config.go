package config

import (
	"os"
	"path/filepath"
)

// PyenvConfig holds all configuration for pyenv-win
type PyenvConfig struct {
	PyenvHome      string
	PyenvParent    string
	CacheDir       string
	VersionsDir    string
	LibexecDir     string
	ShimsDir       string
	WixDir         string
	DBFile         string
	VersionFile    string
	Mirrors        []string
	NexusServer    string
	ForceArch      string
	HTTPProxy      string
	HTTPSProxy     string
}

// New creates a new PyenvConfig with default values
func New() *PyenvConfig {
	pyenvHome := os.Getenv("PYENV")
	if pyenvHome == "" {
		pyenvHome = os.Getenv("PYENV_HOME")
	}
	if pyenvHome == "" {
		home, _ := os.UserHomeDir()
		pyenvHome = filepath.Join(home, ".pyenv", "pyenv-win")
	}

	pyenvParent := filepath.Dir(pyenvHome)

	cfg := &PyenvConfig{
		PyenvHome:   pyenvHome,
		PyenvParent: pyenvParent,
		CacheDir:    filepath.Join(pyenvHome, "install_cache"),
		VersionsDir: filepath.Join(pyenvHome, "versions"),
		LibexecDir:  filepath.Join(pyenvHome, "libexec"),
		ShimsDir:    filepath.Join(pyenvHome, "shims"),
		WixDir:      filepath.Join(pyenvHome, "bin", "WiX"),
		DBFile:      filepath.Join(pyenvHome, ".versions_cache.xml"),
		VersionFile: ".python-version",
		NexusServer: os.Getenv("PYENV_NEXUS_SERVER"),
		ForceArch:   os.Getenv("PYENV_FORCE_ARCH"),
		HTTPProxy:   os.Getenv("http_proxy"),
		HTTPSProxy:  os.Getenv("https_proxy"),
	}

	// Set default mirrors
	mirrorURL := os.Getenv("PYTHON_BUILD_MIRROR_URL")
	if mirrorURL != "" {
		cfg.Mirrors = []string{mirrorURL}
	} else {
		cfg.Mirrors = []string{
			"https://www.python.org/ftp/python",
			"https://downloads.python.org/pypy/versions.json",
			"https://api.github.com/repos/oracle/graalpython/releases",
		}
	}

	return cfg
}

// GetArchPostfix returns the architecture postfix based on system architecture
func (c *PyenvConfig) GetArchPostfix() string {
	arch := c.ForceArch
	if arch == "" {
		arch = os.Getenv("PROCESSOR_ARCHITECTURE")
	}

	switch arch {
	case "AMD64":
		return ""
	case "X86":
		return "-win32"
	case "ARM64":
		return "-arm64"
	default:
		return ""
	}
}

// Is32Bit returns true if running on 32-bit architecture
func (c *PyenvConfig) Is32Bit() bool {
	arch := c.ForceArch
	if arch == "" {
		arch = os.Getenv("PROCESSOR_ARCHITECTURE")
	}
	return arch == "X86"
}
