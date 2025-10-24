package download

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Client handles HTTP downloads
type Client struct {
	httpClient *http.Client
	proxy      string
	xdrEvade   bool // Enable Cortex XDR evasion techniques
}

// TLSConfig contains TLS/SSL configuration options
type TLSConfig struct {
	VerifySSL       bool   // Verify SSL certificates
	CABundle        string // Path to CA certificate bundle
	ClientCert      string // Path to client certificate (for mutual TLS)
	ClientKey       string // Path to client key (for mutual TLS)
	InsecureSkipAll bool   // Skip all verification (dangerous!)
}

// NewClient creates a new download client
func NewClient(httpProxy, httpsProxy string) *Client {
	return NewClientWithTLS(httpProxy, httpsProxy, nil)
}

// NewClientWithTLS creates a new download client with custom TLS configuration
func NewClientWithTLS(httpProxy, httpsProxy string, tlsConfig *TLSConfig) *Client {
	// Build TLS config
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
	}

	// Apply custom TLS settings if provided
	if tlsConfig != nil {
		if tlsConfig.InsecureSkipAll {
			// Dangerous: Skip all verification
			tlsCfg.InsecureSkipVerify = true
		} else if !tlsConfig.VerifySSL {
			// Skip certificate verification
			tlsCfg.InsecureSkipVerify = true
		} else {
			// Load CA bundle if provided
			if tlsConfig.CABundle != "" {
				if caCertPool, err := loadCABundle(tlsConfig.CABundle); err == nil {
					tlsCfg.RootCAs = caCertPool
				}
			}

			// Load client certificate for mutual TLS if provided
			if tlsConfig.ClientCert != "" && tlsConfig.ClientKey != "" {
				if cert, err := tls.LoadX509KeyPair(tlsConfig.ClientCert, tlsConfig.ClientKey); err == nil {
					tlsCfg.Certificates = []tls.Certificate{cert}
				}
			}
		}
	}

	// Use TLS config and connection pooling to appear more like a browser
	transport := &http.Transport{
		Proxy:               nil,
		TLSClientConfig:     tlsCfg,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	// Set proxy if configured
	proxyURL := httpProxy
	if proxyURL == "" {
		proxyURL = httpsProxy
	}

	if proxyURL != "" {
		if proxy, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxy)
		}
	}

	client := &http.Client{
		Timeout:   10 * time.Minute,
		Transport: transport,
	}

	return &Client{
		httpClient: client,
		proxy:      proxyURL,
		xdrEvade:   true, // Enable by default for safety
	}
}

// loadCABundle loads a CA certificate bundle from file
func loadCABundle(path string) (*x509.CertPool, error) {
	caCert, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA bundle: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA bundle")
	}

	return caCertPool, nil
}

// SetXDREvade enables or disables Cortex XDR evasion techniques
func (c *Client) SetXDREvade(enable bool) {
	c.xdrEvade = enable
}

// SetTimeout sets the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// createRequest creates an HTTP request with appropriate headers to avoid detection
func (c *Client) createRequest(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	if c.xdrEvade {
		// Use realistic browser user-agent to avoid detection
		userAgents := []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:122.0) Gecko/20100101 Firefox/122.0",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36 Edg/121.0.0.0",
		}
		req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		req.Header.Set("DNT", "1")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
	}

	return req, nil
}

// xdrSafeDelay adds a small random delay to avoid detection patterns
func (c *Client) xdrSafeDelay() {
	if c.xdrEvade {
		// Random delay between 100ms and 500ms to appear more human-like
		delay := time.Duration(100+rand.Intn(400)) * time.Millisecond
		time.Sleep(delay)
	}
}

// DownloadFile downloads a file from URL to destination
func (c *Client) DownloadFile(url, dest string) error {
	// Add XDR-safe delay before request
	c.xdrSafeDelay()

	req, err := c.createRequest("GET", url)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// FetchText fetches text content from a URL
func (c *Client) FetchText(url string) (string, error) {
	// Add XDR-safe delay before request
	c.xdrSafeDelay()

	req, err := c.createRequest("GET", url)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

// FetchBytes fetches binary content from a URL
func (c *Client) FetchBytes(url string) ([]byte, error) {
	// Add XDR-safe delay before request
	c.xdrSafeDelay()

	req, err := c.createRequest("GET", url)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// DownloadWithProgress downloads a file with progress reporting
func (c *Client) DownloadWithProgress(url, dest string, progress func(current, total int64)) error {
	// Add XDR-safe delay before request
	c.xdrSafeDelay()

	req, err := c.createRequest("GET", url)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Get content length
	contentLength := resp.ContentLength

	// Create progress reader
	reader := &progressReader{
		reader:   resp.Body,
		total:    contentLength,
		progress: progress,
	}

	_, err = io.Copy(out, reader)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

type progressReader struct {
	reader   io.Reader
	current  int64
	total    int64
	progress func(current, total int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)

	if pr.progress != nil {
		pr.progress(pr.current, pr.total)
	}

	return n, err
}
