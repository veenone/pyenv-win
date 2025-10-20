package cache

import (
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Version represents a Python version in the cache
type Version struct {
	XMLName      xml.Name `xml:"version"`
	Code         string   `xml:"code"`
	File         string   `xml:"file"`
	URL          string   `xml:"URL"`
	ZipRootDir   string   `xml:"zipRootDir,omitempty"`
	X64          bool     `xml:"x64,attr"`
	WebInstall   bool     `xml:"webInstall,attr"`
	MSI          bool     `xml:"msi,attr"`
}

// VersionCache represents the versions cache XML
type VersionCache struct {
	XMLName  xml.Name   `xml:"versions"`
	Versions []*Version `xml:"version"`
}

// VersionComponents represents parsed version components
type VersionComponents struct {
	Major      string
	Minor      string
	Patch      string
	Release    string
	RelNumber  string
	Arch       string
	Web        string
	Ext        string
	ZipRoot    string
}

var (
	regexVer     = regexp.MustCompile(`^(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:([a-z]+)(\d*))?$`)
	regexVerArch = regexp.MustCompile(`^(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:([a-z]+)(\d*))?([\.-](?:amd64|arm64|win32))?$`)
	regexFile    = regexp.MustCompile(`^python-(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:([a-z]+)(\d*))?([\.-]amd64)?([\.-]arm64)?(-webinstall)?\.(exe|msi|zip)$`)
)

// LoadVersionsXML loads the versions cache from XML file
func LoadVersionsXML(path string) (*VersionCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cache VersionCache
	err = xml.Unmarshal(data, &cache)
	if err != nil {
		return nil, err
	}

	return &cache, nil
}

// SaveVersionsXML saves the versions cache to XML file
func SaveVersionsXML(path string, cache *VersionCache) error {
	// Sort versions before saving
	sort.Slice(cache.Versions, func(i, j int) bool {
		return CompareVersions(cache.Versions[i].Code, cache.Versions[j].Code)
	})

	output, err := xml.MarshalIndent(cache, "", "\t")
	if err != nil {
		return err
	}

	// Add XML header
	xmlData := []byte(xml.Header + string(output))

	return os.WriteFile(path, xmlData, 0644)
}

// GetVersion retrieves a version by code
func (c *VersionCache) GetVersion(code string) *Version {
	for _, v := range c.Versions {
		if v.Code == code {
			return v
		}
	}
	return nil
}

// AddVersion adds a version to the cache
func (c *VersionCache) AddVersion(v *Version) {
	// Check if version already exists
	for i, existing := range c.Versions {
		if existing.Code == v.Code {
			c.Versions[i] = v
			return
		}
	}
	c.Versions = append(c.Versions, v)
}

// ListVersions returns all version codes
func (c *VersionCache) ListVersions() []string {
	var codes []string
	for _, v := range c.Versions {
		codes = append(codes, v.Code)
	}
	return codes
}

// FindLatestVersion finds the latest version matching a prefix
func (c *VersionCache) FindLatestVersion(prefix, arch string) string {
	var bestMatch *Version

	for _, v := range c.Versions {
		// Check if version starts with prefix
		if !strings.HasPrefix(v.Code, prefix) {
			continue
		}

		// Full match OR prefix plus '.'
		if v.Code != prefix+arch && !strings.HasPrefix(v.Code, prefix+".") {
			continue
		}

		// Parse version
		matches := regexVerArch.FindStringSubmatch(v.Code)
		if len(matches) == 0 {
			continue
		}

		// Skip dev builds (releases like a1, b1, rc1, etc.)
		if matches[4] != "" {
			continue
		}

		// Check architecture matches
		archSuffix := matches[6]
		if archSuffix != arch {
			continue
		}

		// Compare with current best match
		if bestMatch == nil {
			bestMatch = v
		} else if CompareVersions(bestMatch.Code, v.Code) {
			bestMatch = v
		}
	}

	if bestMatch == nil {
		return ""
	}
	return bestMatch.Code
}

// CompareVersions returns true if v1 < v2 (semantic version comparison)
func CompareVersions(v1, v2 string) bool {
	matches1 := regexVerArch.FindStringSubmatch(v1)
	matches2 := regexVerArch.FindStringSubmatch(v2)

	if len(matches1) == 0 || len(matches2) == 0 {
		return v1 < v2
	}

	// Compare major
	if matches1[1] != matches2[1] {
		return parseIntOrZero(matches1[1]) < parseIntOrZero(matches2[1])
	}

	// Compare minor
	if matches1[2] != matches2[2] {
		return parseIntOrZero(matches1[2]) < parseIntOrZero(matches2[2])
	}

	// Compare patch
	if matches1[3] != matches2[3] {
		return parseIntOrZero(matches1[3]) < parseIntOrZero(matches2[3])
	}

	// Compare release (a, b, rc)
	if matches1[4] != matches2[4] {
		// No release > has release
		if matches1[4] == "" && matches2[4] != "" {
			return false
		}
		if matches1[4] != "" && matches2[4] == "" {
			return true
		}
		return matches1[4] < matches2[4]
	}

	// Compare release number
	if matches1[5] != matches2[5] {
		return parseIntOrZero(matches1[5]) < parseIntOrZero(matches2[5])
	}

	// Compare arch
	return matches1[6] < matches2[6]
}

func parseIntOrZero(s string) int {
	var val int
	fmt.Sscanf(s, "%d", &val)
	return val
}

// TryResolveVersion tries to resolve a version prefix to an exact version
func (c *VersionCache) TryResolveVersion(prefix, arch string) string {
	resolved := c.FindLatestVersion(prefix, arch)
	if resolved == "" {
		return prefix
	}
	return resolved
}

// ParseFileName parses a Python installer filename
func ParseFileName(filename string) (*VersionComponents, error) {
	matches := regexFile.FindStringSubmatch(filename)
	if len(matches) == 0 {
		return nil, fmt.Errorf("invalid filename format: %s", filename)
	}

	vc := &VersionComponents{
		Major:     matches[1],
		Minor:     matches[2],
		Patch:     matches[3],
		Release:   matches[4],
		RelNumber: matches[5],
		Ext:       matches[9],
	}

	// Determine architecture
	if matches[6] != "" {
		vc.Arch = "-amd64"
	} else if matches[7] != "" {
		vc.Arch = "-arm64"
	} else {
		vc.Arch = "-win32"
	}

	// Check for web install
	if matches[8] != "" {
		vc.Web = "-webinstall"
	}

	return vc, nil
}

// BuildVersionCode builds a version code from components
func (vc *VersionComponents) BuildVersionCode() string {
	code := vc.Major
	if vc.Minor != "" {
		code += "." + vc.Minor
	}
	if vc.Patch != "" {
		code += "." + vc.Patch
	}
	if vc.Release != "" {
		code += vc.Release
	}
	if vc.RelNumber != "" {
		code += vc.RelNumber
	}

	// Add architecture suffix for non-amd64
	if vc.Arch == "-win32" {
		code += "-win32"
	} else if vc.Arch == "-arm64" {
		code += "-arm"
	}

	return code
}

// Check32Bit converts a version to 32-bit if running on 32-bit system
func Check32Bit(version string, is32Bit bool) string {
	if !is32Bit {
		return version
	}
	if !strings.HasSuffix(version, "-win32") && !strings.HasSuffix(version, "-arm") {
		return version + "-win32"
	}
	return version
}
