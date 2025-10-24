package commands

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/pyenv-win/pyenv-win/internal/cache"
	"github.com/pyenv-win/pyenv-win/internal/config"
	"github.com/pyenv-win/pyenv-win/internal/download"
	"github.com/pyenv-win/pyenv-win/internal/utils"
	"golang.org/x/net/html"
)

var (
	regexJsonURL = regexp.MustCompile(`download_url":\s*"(https://[^\s"]+/(((?:pypy\d+\.\d+-v|graalpy-)(\d+)(?:\.(\d+))?(?:\.(\d+))?-(win64|windows-amd64|windows-aarch64)).zip))"`)
)

// UpdateCommand handles the update command
type UpdateCommand struct {
	cfg    *config.PyenvConfig
	client *download.Client
}

// NewUpdateCommand creates a new update command
func NewUpdateCommand(cfg *config.PyenvConfig) *UpdateCommand {
	return &UpdateCommand{
		cfg:    cfg,
		client: download.NewClient(cfg.HTTPProxy, cfg.HTTPSProxy),
	}
}

// Execute runs the update command
func (u *UpdateCommand) Execute(ignoreErrors bool) error {
	utils.PrintInfo("Mirror: %s", strings.Join(u.cfg.Mirrors, "\n:: [Info] ::  Mirror: "))

	// Load existing cache
	var existingCache *cache.VersionCache
	if utils.FileExists(u.cfg.DBFile) {
		var err error
		existingCache, err = cache.LoadVersionsXML(u.cfg.DBFile)
		if err != nil {
			return fmt.Errorf("failed to load existing cache: %w", err)
		}
		utils.PrintInfo("Loaded %d existing versions from cache", len(existingCache.Versions))
	} else {
		existingCache = &cache.VersionCache{}
	}

	// Scan mirrors for new versions
	allVersions := make(map[string]*cache.Version)
	pageCount := 0

	// Add existing versions to map
	for _, v := range existingCache.Versions {
		allVersions[v.File] = v
	}

	for _, mirror := range u.cfg.Mirrors {
		versions, pages, err := u.scanMirror(mirror, ignoreErrors)
		if err != nil {
			if ignoreErrors {
				utils.PrintError("Failed to scan mirror %s: %v", mirror, err)
				continue
			}
			return err
		}

		pageCount += pages

		// Merge versions (new versions override existing)
		for _, v := range versions {
			allVersions[v.File] = v
		}
	}

	// Convert map to slice
	newCache := &cache.VersionCache{}
	for _, v := range allVersions {
		newCache.Versions = append(newCache.Versions, v)
	}

	utils.PrintInfo("Merged to %d total versions", len(newCache.Versions))

	// Save cache
	if err := cache.SaveVersionsXML(u.cfg.DBFile, newCache); err != nil {
		return fmt.Errorf("failed to save cache: %w", err)
	}

	utils.PrintInfo("Scanned %d pages and found %d installers.", pageCount, len(newCache.Versions))
	return nil
}

func (u *UpdateCommand) scanMirror(mirrorURL string, ignoreErrors bool) ([]*cache.Version, int, error) {
	// Check if URL is JSON (for PyPy and GraalPy)
	if strings.HasSuffix(mirrorURL, ".json") || strings.Contains(mirrorURL, "/releases") {
		return u.scanJSONMirror(mirrorURL, ignoreErrors)
	}

	return u.scanHTMLMirror(mirrorURL, ignoreErrors)
}

func (u *UpdateCommand) scanJSONMirror(mirrorURL string, ignoreErrors bool) ([]*cache.Version, int, error) {
	responseText, err := u.client.FetchText(mirrorURL)
	if err != nil {
		if ignoreErrors {
			return nil, 0, nil
		}
		return nil, 0, err
	}

	versions := []*cache.Version{}

	// Find all download URLs using regex
	matches := regexJsonURL.FindAllStringSubmatch(responseText, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		url := match[1]
		filename := match[2]
		zipRoot := match[3]
		_ = match[4] // major
		_ = match[5] // minor
		_ = match[6] // patch
		arch := match[7]

		// Determine architecture
		x64 := false
		if arch == "win64" || arch == "windows-amd64" {
			x64 = true
		} else if arch == "windows-aarch64" {
			// ARM architecture
			x64 = false
		}

		// Build version code
		code := zipRoot

		version := &cache.Version{
			Code:       code,
			File:       filename,
			URL:        url,
			ZipRootDir: zipRoot,
			X64:        x64,
			WebInstall: false,
			MSI:        false,
		}

		versions = append(versions, version)
	}

	return versions, 1, nil
}

func (u *UpdateCommand) scanHTMLMirror(mirrorURL string, ignoreErrors bool) ([]*cache.Version, int, error) {
	responseText, err := u.client.FetchText(mirrorURL)
	if err != nil {
		if ignoreErrors {
			return nil, 0, nil
		}
		return nil, 0, err
	}

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(responseText))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse HTML: %w", err)
	}

	versions := []*cache.Version{}
	pageCount := 1

	// Extract all links
	links := u.extractLinks(doc, mirrorURL)

	for _, link := range links {
		// Check if link is a version directory (e.g., "3.10.0/")
		if strings.HasSuffix(link.Text, "/") && isVersionDir(link.Text) {
			// Scan subdirectory
			subVersions, subPages, err := u.scanHTMLMirror(link.Href, ignoreErrors)
			if err != nil {
				if ignoreErrors {
					continue
				}
				return nil, 0, err
			}
			versions = append(versions, subVersions...)
			pageCount += subPages
		} else {
			// Check if link is an installer file
			version := u.parseInstallerLink(link)
			if version != nil {
				versions = append(versions, version)
			}
		}
	}

	return versions, pageCount, nil
}

type Link struct {
	Href string
	Text string
}

func (u *UpdateCommand) extractLinks(n *html.Node, baseURL string) []Link {
	var links []Link

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			var href, text string
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href = attr.Val
				}
			}

			// Get link text
			if n.FirstChild != nil {
				text = n.FirstChild.Data
			}

			// Make absolute URL
			if href != "" && !strings.HasPrefix(href, "http") {
				if strings.HasPrefix(href, "/") {
					// Absolute path
					parts := strings.SplitN(baseURL, "/", 4)
					if len(parts) >= 3 {
						href = parts[0] + "//" + parts[2] + href
					}
				} else {
					// Relative path
					if !strings.HasSuffix(baseURL, "/") {
						baseURL += "/"
					}
					href = baseURL + href
				}
			}

			if href != "" {
				links = append(links, Link{Href: href, Text: text})
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(n)
	return links
}

func isVersionDir(text string) bool {
	// Match version directory patterns like "3.10.0/"
	matched, _ := regexp.MatchString(`^\d+\.\d+(\.\d+)?/$`, text)
	return matched
}

func (u *UpdateCommand) parseInstallerLink(link Link) *cache.Version {
	filename := link.Text

	// Parse filename using regex
	vc, err := cache.ParseFileName(filename)
	if err != nil {
		return nil
	}

	// Build version code
	code := vc.BuildVersionCode()

	// Determine flags
	x64 := vc.Arch == "-amd64" || vc.Arch == "-arm64"
	msi := vc.Ext == "msi"
	web := vc.Web != ""

	version := &cache.Version{
		Code:       code,
		File:       filename,
		URL:        link.Href,
		X64:        x64,
		WebInstall: web,
		MSI:        msi,
	}

	// Set ZipRootDir for zip files
	if vc.Ext == "zip" && vc.ZipRoot != "" {
		version.ZipRootDir = vc.ZipRoot
	}

	return version
}

// PyPyRelease represents a PyPy release from the JSON API
type PyPyRelease struct {
	PyPyVersion   string `json:"pypy_version"`
	PythonVersion string `json:"python_version"`
	Files         []struct {
		Filename     string `json:"filename"`
		DownloadURL  string `json:"download_url"`
		Architecture string `json:"architecture"`
		Platform     string `json:"platform"`
	} `json:"files"`
}

func (u *UpdateCommand) scanPyPyJSON(mirrorURL string) ([]*cache.Version, error) {
	responseText, err := u.client.FetchText(mirrorURL)
	if err != nil {
		return nil, err
	}

	var releases []PyPyRelease
	if err := json.Unmarshal([]byte(responseText), &releases); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	versions := []*cache.Version{}

	for _, release := range releases {
		for _, file := range release.Files {
			// Only include Windows versions
			if file.Platform != "win64" && file.Platform != "win32" {
				continue
			}

			// Only include zip files
			if !strings.HasSuffix(file.Filename, ".zip") {
				continue
			}

			// Parse filename to get version info
			vc, err := cache.ParseFileName(file.Filename)
			if err != nil {
				continue
			}

			code := vc.BuildVersionCode()

			version := &cache.Version{
				Code:       code,
				File:       file.Filename,
				URL:        file.DownloadURL,
				ZipRootDir: vc.ZipRoot,
				X64:        file.Platform == "win64",
				WebInstall: false,
				MSI:        false,
			}

			versions = append(versions, version)
		}
	}

	return versions, nil
}
