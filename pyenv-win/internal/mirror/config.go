// +build mirror

package mirror

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pyenv-win/pyenv-win/internal/utils"
)

// MirrorConfig represents the mirror configuration
type MirrorConfig struct {
	// Nexus server configuration
	Nexus NexusConfig `json:"nexus"`

	// Download settings
	Download DownloadConfig `json:"download"`

	// Mirror behavior settings
	Behavior BehaviorConfig `json:"behavior"`
}

// NexusConfig contains Nexus server settings
type NexusConfig struct {
	URL             string `json:"url"`                      // Nexus server URL (required)
	User            string `json:"user"`                     // Username for authentication
	Password        string `json:"password"`                 // Password for authentication
	Repo            string `json:"repo"`                     // Repository name
	UseToken        bool   `json:"use_token"`                // Use token authentication instead of user/password
	Token           string `json:"token,omitempty"`          // API token (if use_token is true)
	VerifySSL       bool   `json:"verify_ssl"`               // Verify SSL certificates (default: true)
	CABundle        string `json:"ca_bundle,omitempty"`      // Path to CA certificate bundle file
	ClientCert      string `json:"client_cert,omitempty"`    // Path to client certificate file (for mutual TLS)
	ClientKey       string `json:"client_key,omitempty"`     // Path to client private key file (for mutual TLS)
	InsecureSkipAll bool   `json:"insecure_skip_all"`        // Skip all SSL verification (dangerous, use only for testing)
}

// DownloadConfig contains download behavior settings
type DownloadConfig struct {
	CacheDir        string `json:"cache_dir"`          // Directory to cache downloads
	MaxRetries      int    `json:"max_retries"`        // Maximum retry attempts
	RetryDelay      int    `json:"retry_delay_ms"`     // Delay between retries in milliseconds
	Timeout         int    `json:"timeout_seconds"`    // HTTP timeout in seconds
	ChunkSize       int    `json:"chunk_size_mb"`      // Download chunk size in MB
	VerifyChecksum  bool   `json:"verify_checksum"`    // Verify checksums after download
}

// BehaviorConfig contains mirror behavior settings
type BehaviorConfig struct {
	SkipExisting    bool     `json:"skip_existing"`      // Skip versions that exist in Nexus
	DeleteAfterSync bool     `json:"delete_after_sync"`  // Delete local files after successful upload
	Parallel        int      `json:"parallel_uploads"`   // Number of parallel uploads (0 = sequential)
	DelayBetween    int      `json:"delay_between_ms"`   // Delay between uploads in milliseconds
	IncludePatterns []string `json:"include_patterns"`   // Version patterns to include (e.g., "3.12.*")
	ExcludePatterns []string `json:"exclude_patterns"`   // Version patterns to exclude (e.g., "*rc*")
	ArchFilter      string   `json:"arch_filter"`        // Architecture filter: "amd64", "win32", or "" for all
	StableOnly      bool     `json:"stable_only"`        // Only include stable versions (exclude alpha/beta/rc)
}

// DefaultMirrorConfig returns a config with sensible defaults
func DefaultMirrorConfig() *MirrorConfig {
	return &MirrorConfig{
		Nexus: NexusConfig{
			URL:             "",
			User:            "admin",
			Password:        "",
			Repo:            "python-binaries",
			UseToken:        false,
			VerifySSL:       true,  // Secure by default
			CABundle:        "",
			ClientCert:      "",
			ClientKey:       "",
			InsecureSkipAll: false,
		},
		Download: DownloadConfig{
			CacheDir:       "",  // Will be set to pyenv cache dir
			MaxRetries:     3,
			RetryDelay:     1000, // 1 second
			Timeout:        600,  // 10 minutes
			ChunkSize:      10,   // 10 MB chunks
			VerifyChecksum: false,
		},
		Behavior: BehaviorConfig{
			SkipExisting:    true,
			DeleteAfterSync: false,
			Parallel:        0,  // Sequential by default
			DelayBetween:    2000, // 2 seconds
			IncludePatterns: []string{},
			ExcludePatterns: []string{},
			ArchFilter:      "",    // All architectures by default
			StableOnly:      false, // Include all versions by default
		},
	}
}

// LoadMirrorConfig loads mirror configuration from file or environment
func LoadMirrorConfig(configPath string) (*MirrorConfig, error) {
	config := DefaultMirrorConfig()

	// Try to load from config file first
	if configPath != "" {
		if err := loadFromFile(config, configPath); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	} else {
		// Try default locations
		defaultPaths := []string{
			"mirror.json",
			".mirror.json",
			filepath.Join(os.Getenv("PYENV"), "mirror.json"),
			filepath.Join(os.Getenv("PYENV_HOME"), "mirror.json"),
		}

		// Try to load from default locations
		loaded := false
		for _, path := range defaultPaths {
			if path != "" && utils.FileExists(path) {
				if err := loadFromFile(config, path); err == nil {
					loaded = true
					break
				}
			}
		}

		// If no config file found, try environment variables
		if !loaded {
			loadFromEnv(config)
		}
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// loadFromFile loads configuration from a JSON file
func loadFromFile(config *MirrorConfig, path string) error {
	data, err := utils.ReadFile(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return nil
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(config *MirrorConfig) {
	if url := os.Getenv("NEXUS_URL"); url != "" {
		config.Nexus.URL = url
	}
	if user := os.Getenv("NEXUS_USER"); user != "" {
		config.Nexus.User = user
	}
	if password := os.Getenv("NEXUS_PASSWORD"); password != "" {
		config.Nexus.Password = password
	}
	if repo := os.Getenv("NEXUS_REPO"); repo != "" {
		config.Nexus.Repo = repo
	}
	if token := os.Getenv("NEXUS_TOKEN"); token != "" {
		config.Nexus.Token = token
		config.Nexus.UseToken = true
	}
}

// Validate validates the mirror configuration
func (mc *MirrorConfig) Validate() error {
	// Validate Nexus config
	if mc.Nexus.URL == "" {
		return fmt.Errorf("nexus.url is required")
	}

	if mc.Nexus.UseToken {
		if mc.Nexus.Token == "" {
			return fmt.Errorf("nexus.token is required when use_token is true")
		}
	} else {
		if mc.Nexus.Password == "" {
			return fmt.Errorf("nexus.password is required when use_token is false")
		}
	}

	if mc.Nexus.Repo == "" {
		return fmt.Errorf("nexus.repo is required")
	}

	// Validate SSL config
	if mc.Nexus.CABundle != "" && !fileExists(mc.Nexus.CABundle) {
		return fmt.Errorf("nexus.ca_bundle file not found: %s", mc.Nexus.CABundle)
	}
	if mc.Nexus.ClientCert != "" && !fileExists(mc.Nexus.ClientCert) {
		return fmt.Errorf("nexus.client_cert file not found: %s", mc.Nexus.ClientCert)
	}
	if mc.Nexus.ClientKey != "" && !fileExists(mc.Nexus.ClientKey) {
		return fmt.Errorf("nexus.client_key file not found: %s", mc.Nexus.ClientKey)
	}
	if (mc.Nexus.ClientCert != "" && mc.Nexus.ClientKey == "") ||
		(mc.Nexus.ClientCert == "" && mc.Nexus.ClientKey != "") {
		return fmt.Errorf("nexus.client_cert and nexus.client_key must both be set for mutual TLS")
	}

	// Validate Download config
	if mc.Download.MaxRetries < 0 {
		return fmt.Errorf("download.max_retries must be >= 0")
	}
	if mc.Download.RetryDelay < 0 {
		return fmt.Errorf("download.retry_delay_ms must be >= 0")
	}
	if mc.Download.Timeout <= 0 {
		return fmt.Errorf("download.timeout_seconds must be > 0")
	}
	if mc.Download.ChunkSize <= 0 {
		return fmt.Errorf("download.chunk_size_mb must be > 0")
	}

	// Validate Behavior config
	if mc.Behavior.Parallel < 0 {
		return fmt.Errorf("behavior.parallel_uploads must be >= 0")
	}
	if mc.Behavior.DelayBetween < 0 {
		return fmt.Errorf("behavior.delay_between_ms must be >= 0")
	}

	return nil
}

// SaveToFile saves the configuration to a JSON file
func (mc *MirrorConfig) SaveToFile(path string) error {
	data, err := json.MarshalIndent(mc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := utils.WriteFile(path, data); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetAuth returns the authentication method and credentials
func (nc *NexusConfig) GetAuth() (authType string, user string, credential string) {
	if nc.UseToken {
		return "token", nc.User, nc.Token
	}
	return "basic", nc.User, nc.Password
}

// GetTLSConfig creates a TLS configuration from the Nexus config
func (nc *NexusConfig) GetTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
	}

	// Apply SSL verification settings
	if nc.InsecureSkipAll {
		// Dangerous: Skip all verification
		tlsConfig.InsecureSkipVerify = true
		return tlsConfig, nil
	}

	if !nc.VerifySSL {
		// Skip certificate verification
		tlsConfig.InsecureSkipVerify = true
		return tlsConfig, nil
	}

	// Load CA bundle if provided
	if nc.CABundle != "" {
		caCert, err := os.ReadFile(nc.CABundle)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA bundle: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA bundle")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate for mutual TLS if provided
	if nc.ClientCert != "" && nc.ClientKey != "" {
		cert, err := tls.LoadX509KeyPair(nc.ClientCert, nc.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// ShouldIncludeVersion checks if a version should be included based on patterns
func (bc *BehaviorConfig) ShouldIncludeVersion(version string) bool {
	// Check architecture filter
	if bc.ArchFilter != "" {
		if bc.ArchFilter == "amd64" || bc.ArchFilter == "64bit" {
			// Must be amd64 (not win32)
			if contains(version, "win32") {
				return false
			}
		} else if bc.ArchFilter == "win32" || bc.ArchFilter == "32bit" {
			// Must be win32
			if !contains(version, "win32") {
				return false
			}
		}
	}

	// Check stable version filter
	if bc.StableOnly {
		// Exclude alpha, beta, rc versions
		lowerVersion := toLower(version)
		if contains(lowerVersion, "alpha") || contains(lowerVersion, "beta") ||
			contains(lowerVersion, "rc") || contains(lowerVersion, "a") && !contains(lowerVersion, "amd64") {
			return false
		}
	}

	// If no patterns specified, include all
	if len(bc.IncludePatterns) == 0 && len(bc.ExcludePatterns) == 0 {
		return true
	}

	// Check exclude patterns first
	for _, pattern := range bc.ExcludePatterns {
		if matchPattern(pattern, version) {
			return false
		}
	}

	// If include patterns specified, version must match at least one
	if len(bc.IncludePatterns) > 0 {
		for _, pattern := range bc.IncludePatterns {
			if matchPattern(pattern, version) {
				return true
			}
		}
		return false
	}

	return true
}

// matchPattern matches a version against a pattern (supports * wildcard)
func matchPattern(pattern, version string) bool {
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}

	// Exact match
	if pattern == version {
		return true
	}

	// Wildcard matching
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(version) >= len(prefix) && version[:len(prefix)] == prefix
	}

	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		return len(version) >= len(suffix) && version[len(version)-len(suffix):] == suffix
	}

	// Contains match for patterns like "*rc*"
	if len(pattern) > 2 && pattern[0] == '*' && pattern[len(pattern)-1] == '*' {
		substr := pattern[1 : len(pattern)-1]
		return contains(version, substr)
	}

	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		indexOfSubstring(s, substr) >= 0
}

// indexOfSubstring finds the index of a substring
func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// toLower converts a string to lowercase
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + ('a' - 'A')
		} else {
			result[i] = c
		}
	}
	return string(result)
}
