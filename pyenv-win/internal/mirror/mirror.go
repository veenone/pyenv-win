// +build mirror

package mirror

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pyenv-win/pyenv-win/internal/cache"
	"github.com/pyenv-win/pyenv-win/internal/config"
	"github.com/pyenv-win/pyenv-win/internal/download"
	"github.com/pyenv-win/pyenv-win/internal/utils"
)

// MirrorPlugin handles mirroring Python binaries from python.org to Nexus
type MirrorPlugin struct {
	cfg         *config.PyenvConfig
	mirrorCfg   *MirrorConfig
	downloadDir string
	client      *download.Client
}

// NewMirrorPlugin creates a new mirroring plugin
func NewMirrorPlugin(cfg *config.PyenvConfig, configPath string) (*MirrorPlugin, error) {
	// Load mirror configuration
	mirrorCfg, err := LoadMirrorConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load mirror configuration: %w\n\nPlease create a mirror.json config file or set environment variables.\nRun 'pyenv mirror init' to create a sample config file.", err)
	}

	// Set download directory
	downloadDir := mirrorCfg.Download.CacheDir
	if downloadDir == "" {
		downloadDir = filepath.Join(cfg.CacheDir, "mirror_downloads")
	}
	if err := utils.EnsureDir(downloadDir); err != nil {
		return nil, fmt.Errorf("failed to create mirror download directory: %w", err)
	}

	// Create TLS configuration from mirror config
	tlsConfig := &download.TLSConfig{
		VerifySSL:       mirrorCfg.Nexus.VerifySSL,
		CABundle:        mirrorCfg.Nexus.CABundle,
		ClientCert:      mirrorCfg.Nexus.ClientCert,
		ClientKey:       mirrorCfg.Nexus.ClientKey,
		InsecureSkipAll: mirrorCfg.Nexus.InsecureSkipAll,
	}

	// Create download client with TLS config
	client := download.NewClientWithTLS(cfg.HTTPProxy, cfg.HTTPSProxy, tlsConfig)

	// Configure client timeout
	client.SetTimeout(time.Duration(mirrorCfg.Download.Timeout) * time.Second)

	// Warn if SSL verification is disabled
	if !mirrorCfg.Nexus.VerifySSL || mirrorCfg.Nexus.InsecureSkipAll {
		utils.PrintError("WARNING: SSL certificate verification is disabled!")
		utils.PrintError("This is insecure and should only be used for testing.")
	}

	return &MirrorPlugin{
		cfg:         cfg,
		mirrorCfg:   mirrorCfg,
		downloadDir: downloadDir,
		client:      client,
	}, nil
}

// InitConfig creates a sample mirror configuration file
func InitConfig(path string) error {
	if path == "" {
		path = "mirror.json"
	}

	// Check if file already exists
	if utils.FileExists(path) {
		return fmt.Errorf("config file already exists: %s\nUse --force to overwrite", path)
	}

	// Create default config
	cfg := DefaultMirrorConfig()

	// Save to file
	if err := cfg.SaveToFile(path); err != nil {
		return fmt.Errorf("failed to save config file: %w", err)
	}

	utils.PrintInfo("Created mirror configuration file: %s", path)
	utils.PrintInfo("\nPlease edit this file and set:")
	utils.PrintInfo("  - nexus.url: Your Nexus server URL")
	utils.PrintInfo("  - nexus.password: Your Nexus password")
	utils.PrintInfo("  - nexus.ca_bundle: Path to CA certificate (for HTTPS with self-signed certs)")
	utils.PrintInfo("  - Other settings as needed")
	utils.PrintInfo("\nFor CA certificate setup, see: CA_CERT_SETUP.md")
	utils.PrintInfo("\nThen run: pyenv mirror all")

	return nil
}

// MirrorAll downloads all Python binaries from cache and uploads to Nexus
func (mp *MirrorPlugin) MirrorAll() error {
	utils.PrintInfo("Starting Python binary mirroring to Nexus...")

	// Load version cache
	versionCache, err := cache.LoadVersionsXML(mp.cfg.DBFile)
	if err != nil {
		return fmt.Errorf("failed to load version cache: %w", err)
	}

	total := len(versionCache.Versions)
	utils.PrintInfo("Found %d Python versions to mirror", total)

	successCount := 0
	failCount := 0
	skipCount := 0

	for i, version := range versionCache.Versions {
		// Check if version should be included based on patterns
		if !mp.mirrorCfg.Behavior.ShouldIncludeVersion(version.Code) {
			skipCount++
			continue
		}

		utils.PrintInfo("[%d/%d] Processing %s...", i+1, total, version.Code)

		// Check if already exists in Nexus (if skip_existing is enabled)
		if mp.mirrorCfg.Behavior.SkipExisting && mp.existsInNexus(*version) {
			utils.PrintInfo("  Already exists in Nexus, skipping")
			skipCount++
			continue
		}

		// Download from Python.org
		localPath, err := mp.downloadVersion(*version)
		if err != nil {
			utils.PrintError("  Failed to download: %v", err)
			failCount++
			continue
		}

		// Upload to Nexus
		if err := mp.uploadToNexus(*version, localPath); err != nil {
			utils.PrintError("  Failed to upload to Nexus: %v", err)
			failCount++
			// Clean up downloaded file
			os.Remove(localPath)
			continue
		}

		utils.PrintInfo("  Successfully mirrored to Nexus")
		successCount++

		// Clean up downloaded file if configured
		if mp.mirrorCfg.Behavior.DeleteAfterSync {
			os.Remove(localPath)
		}

		// Add configured delay between uploads
		if mp.mirrorCfg.Behavior.DelayBetween > 0 {
			time.Sleep(time.Duration(mp.mirrorCfg.Behavior.DelayBetween) * time.Millisecond)
		}
	}

	utils.PrintInfo("\nMirroring complete:")
	utils.PrintInfo("  Success: %d", successCount)
	utils.PrintInfo("  Skipped: %d", skipCount)
	utils.PrintInfo("  Failed:  %d", failCount)

	return nil
}

// expandVersionPattern expands version patterns like "3.8-win32" to latest "3.8.x-win32"
func (mp *MirrorPlugin) expandVersionPattern(pattern string, versionCache *cache.VersionCache) []string {
	var matchedVersions []string

	// Check if it's a pattern that needs expansion
	// Examples: "3" -> latest 3.x, "3.14-win32" -> latest 3.14.x-win32
	for i := range versionCache.Versions {
		v := versionCache.Versions[i]

		// Check if version matches the pattern
		if mp.matchesVersionPattern(v.Code, pattern) {
			// Apply filters from config
			if !mp.mirrorCfg.Behavior.ShouldIncludeVersion(v.Code) {
				continue
			}
			matchedVersions = append(matchedVersions, v.Code)
		}
	}

	// If we found matches, return the latest one
	if len(matchedVersions) > 0 {
		latest := mp.getLatestVersion(matchedVersions)
		utils.PrintInfo("  Pattern '%s' expanded to '%s'", pattern, latest)
		return []string{latest}
	}

	// No match, return the original pattern
	return []string{pattern}
}

// matchesVersionPattern checks if a version matches a pattern
func (mp *MirrorPlugin) matchesVersionPattern(version, pattern string) bool {
	// First, check if architecture suffix matches
	// Pattern: "3.14-win32" should match version: "3.14.0-win32"
	// Pattern: "3.14-amd64" should match version: "3.14.0-amd64"
	// Pattern: "3.14" should match both "3.14.0-win32" and "3.14.0-amd64"

	var patternArch, versionArch string
	patternBase := pattern
	versionBase := version

	// Extract architecture from pattern
	if strings.HasSuffix(pattern, "-win32") {
		patternArch = "-win32"
		patternBase = strings.TrimSuffix(pattern, "-win32")
	} else if strings.HasSuffix(pattern, "-amd64") {
		patternArch = "-amd64"
		patternBase = strings.TrimSuffix(pattern, "-amd64")
	} else if strings.HasSuffix(pattern, "-arm64") {
		patternArch = "-arm64"
		patternBase = strings.TrimSuffix(pattern, "-arm64")
	}

	// Extract architecture from version
	if strings.HasSuffix(version, "-win32") {
		versionArch = "-win32"
		versionBase = strings.TrimSuffix(version, "-win32")
	} else if strings.HasSuffix(version, "-amd64") {
		versionArch = "-amd64"
		versionBase = strings.TrimSuffix(version, "-amd64")
	} else if strings.HasSuffix(version, "-arm64") {
		versionArch = "-arm64"
		versionBase = strings.TrimSuffix(version, "-arm64")
	}

	// If pattern has architecture, version must match exactly
	if patternArch != "" && patternArch != versionArch {
		return false
	}

	// Split version and pattern by dots
	versionParts := strings.Split(versionBase, ".")
	patternParts := strings.Split(patternBase, ".")

	// Pattern must match the beginning of version
	if len(patternParts) > len(versionParts) {
		return false
	}

	for i, part := range patternParts {
		if versionParts[i] != part {
			return false
		}
	}

	return true
}

// getLatestVersion returns the latest version from a list
func (mp *MirrorPlugin) getLatestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}
	if len(versions) == 1 {
		return versions[0]
	}

	// Simple version comparison - assumes versions are in format "x.y.z"
	latest := versions[0]
	for _, v := range versions[1:] {
		if mp.compareVersions(v, latest) > 0 {
			latest = v
		}
	}
	return latest
}

// compareVersions compares two version strings
// Returns: -1 if v1 < v2, 0 if equal, 1 if v1 > v2
func (mp *MirrorPlugin) compareVersions(v1, v2 string) int {
	// Remove architecture suffix for comparison
	v1 = strings.TrimSuffix(strings.TrimSuffix(v1, "-win32"), "-amd64")
	v2 = strings.TrimSuffix(strings.TrimSuffix(v2, "-win32"), "-amd64")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var num1, num2 int

		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &num1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &num2)
		}

		if num1 < num2 {
			return -1
		}
		if num1 > num2 {
			return 1
		}
	}

	return 0
}

// MirrorVersions downloads and uploads specific Python versions
func (mp *MirrorPlugin) MirrorVersions(versions []string) error {
	utils.PrintInfo("Starting selective Python binary mirroring to Nexus...")

	// Load version cache
	versionCache, err := cache.LoadVersionsXML(mp.cfg.DBFile)
	if err != nil {
		return fmt.Errorf("failed to load version cache: %w", err)
	}

	// Expand version patterns
	expandedVersions := []string{}
	for _, pattern := range versions {
		expanded := mp.expandVersionPattern(pattern, versionCache)
		expandedVersions = append(expandedVersions, expanded...)
	}

	successCount := 0
	failCount := 0

	for _, versionCode := range expandedVersions {
		utils.PrintInfo("Processing %s...", versionCode)

		// Find version in cache
		var targetVersion *cache.Version
		arch := mp.cfg.GetArchPostfix()
		for i := range versionCache.Versions {
			v := versionCache.Versions[i]
			if v.Code == versionCode || v.Code == versionCode+arch {
				targetVersion = v
				break
			}
		}

		if targetVersion == nil {
			utils.PrintError("  Version not found in cache")
			failCount++
			continue
		}

		// Check if already exists in Nexus (if skip_existing is enabled)
		if mp.mirrorCfg.Behavior.SkipExisting && mp.existsInNexus(*targetVersion) {
			utils.PrintInfo("  Already exists in Nexus, skipping")
			continue
		}

		// Download from Python.org
		localPath, err := mp.downloadVersion(*targetVersion)
		if err != nil {
			utils.PrintError("  Failed to download: %v", err)
			failCount++
			continue
		}

		// Upload to Nexus
		if err := mp.uploadToNexus(*targetVersion, localPath); err != nil {
			utils.PrintError("  Failed to upload to Nexus: %v", err)
			failCount++
			os.Remove(localPath)
			continue
		}

		utils.PrintInfo("  Successfully mirrored to Nexus")
		successCount++

		// Clean up downloaded file if configured
		if mp.mirrorCfg.Behavior.DeleteAfterSync {
			os.Remove(localPath)
		}
	}

	utils.PrintInfo("\nMirroring complete:")
	utils.PrintInfo("  Success: %d", successCount)
	utils.PrintInfo("  Failed:  %d", failCount)

	return nil
}

// downloadVersion downloads a Python version from python.org
func (mp *MirrorPlugin) downloadVersion(version cache.Version) (string, error) {
	localPath := filepath.Join(mp.downloadDir, version.File)

	utils.PrintInfo("  Downloading from %s", version.URL)

	// Track last reported progress to reduce output
	lastReportedPercent := -1.0

	// Use download client with progress
	err := mp.client.DownloadWithProgress(version.URL, localPath, func(current, total int64) {
		if total > 0 {
			percent := float64(current) / float64(total) * 100
			// Only report progress every 10%
			if percent-lastReportedPercent >= 10.0 || percent >= 99.9 {
				utils.PrintInfo("    Progress: %.0f%%", percent)
				lastReportedPercent = percent
			}
		}
	})

	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	return localPath, nil
}

// createNexusHTTPClient creates an HTTP client with TLS configuration for Nexus
func (mp *MirrorPlugin) createNexusHTTPClient(timeout time.Duration) (*http.Client, error) {
	tlsConfig, err := mp.mirrorCfg.Nexus.GetTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}

// getVersionDirectory returns the directory name for a version (without architecture suffix)
func (mp *MirrorPlugin) getVersionDirectory(versionCode string) string {
	// Remove architecture suffixes to get base version
	// 3.9.13-win32 -> 3.9.13
	// 3.9.13-amd64 -> 3.9.13
	versionDir := strings.TrimSuffix(versionCode, "-win32")
	versionDir = strings.TrimSuffix(versionDir, "-amd64")
	versionDir = strings.TrimSuffix(versionDir, "-arm64")
	return versionDir
}

// uploadToNexus uploads a file to Nexus repository
func (mp *MirrorPlugin) uploadToNexus(version cache.Version, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get version directory (without architecture suffix)
	// Both win32 and amd64 versions go in the same directory
	versionDir := mp.getVersionDirectory(version.Code)

	// Construct Nexus upload URL
	// Format: {nexusURL}/repository/{repo}/{versionDir}/{filename}
	// Example: https://nexus.com/repository/python-binaries/3.9.13/python-3.9.13.exe
	//          https://nexus.com/repository/python-binaries/3.9.13/python-3.9.13-amd64.exe
	uploadURL := fmt.Sprintf("%s/repository/%s/%s/%s",
		strings.TrimRight(mp.mirrorCfg.Nexus.URL, "/"),
		mp.mirrorCfg.Nexus.Repo,
		versionDir,
		version.File)

	utils.PrintInfo("  Uploading to %s", uploadURL)

	req, err := http.NewRequest("PUT", uploadURL, file)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication based on config
	authType, user, credential := mp.mirrorCfg.Nexus.GetAuth()
	if authType == "token" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", credential))
	} else {
		req.SetBasicAuth(user, credential)
	}

	// Set content type based on file extension
	ext := strings.ToLower(filepath.Ext(version.File))
	switch ext {
	case ".exe":
		req.Header.Set("Content-Type", "application/x-msdownload")
	case ".msi":
		req.Header.Set("Content-Type", "application/x-msi")
	case ".zip":
		req.Header.Set("Content-Type", "application/zip")
	default:
		req.Header.Set("Content-Type", "application/octet-stream")
	}

	// Get file size
	fileInfo, _ := file.Stat()
	req.ContentLength = fileInfo.Size()

	// Create HTTP client with TLS configuration
	client, err := mp.createNexusHTTPClient(30 * time.Minute)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// existsInNexus checks if a version already exists in Nexus
func (mp *MirrorPlugin) existsInNexus(version cache.Version) bool {
	// Get version directory (without architecture suffix)
	versionDir := mp.getVersionDirectory(version.Code)

	// Construct Nexus API URL to check existence
	checkURL := fmt.Sprintf("%s/repository/%s/%s/%s",
		strings.TrimRight(mp.mirrorCfg.Nexus.URL, "/"),
		mp.mirrorCfg.Nexus.Repo,
		versionDir,
		version.File)

	req, err := http.NewRequest("HEAD", checkURL, nil)
	if err != nil {
		return false
	}

	// Set authentication based on config
	authType, user, credential := mp.mirrorCfg.Nexus.GetAuth()
	if authType == "token" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", credential))
	} else {
		req.SetBasicAuth(user, credential)
	}

	// Create HTTP client with TLS configuration
	client, err := mp.createNexusHTTPClient(10 * time.Second)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GenerateNexusMapping creates a JSON mapping file for offline usage
func (mp *MirrorPlugin) GenerateNexusMapping() error {
	utils.PrintInfo("Generating Nexus mapping file...")

	// Load version cache
	versionCache, err := cache.LoadVersionsXML(mp.cfg.DBFile)
	if err != nil {
		return fmt.Errorf("failed to load version cache: %w", err)
	}

	// Create mapping
	mapping := make(map[string]string)
	for _, version := range versionCache.Versions {
		// Get version directory (without architecture suffix)
		versionDir := mp.getVersionDirectory(version.Code)

		// Map original URL to Nexus URL
		nexusURL := fmt.Sprintf("%s/repository/%s/%s/%s",
			strings.TrimRight(mp.mirrorCfg.Nexus.URL, "/"),
			mp.mirrorCfg.Nexus.Repo,
			versionDir,
			version.File)
		mapping[version.URL] = nexusURL
	}

	// Write to JSON file
	mappingFile := filepath.Join(mp.cfg.PyenvHome, "nexus_mapping.json")
	data, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	if err := utils.WriteFile(mappingFile, data); err != nil {
		return fmt.Errorf("failed to write mapping file: %w", err)
	}

	utils.PrintInfo("Nexus mapping file created: %s", mappingFile)
	utils.PrintInfo("Total mappings: %d", len(mapping))

	return nil
}
