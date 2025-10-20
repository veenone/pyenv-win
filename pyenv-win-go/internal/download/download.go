package download

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Client handles HTTP downloads
type Client struct {
	httpClient *http.Client
	proxy      string
}

// NewClient creates a new download client
func NewClient(httpProxy, httpsProxy string) *Client {
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	// Set proxy if configured
	proxyURL := httpProxy
	if proxyURL == "" {
		proxyURL = httpsProxy
	}

	if proxyURL != "" {
		if proxy, err := url.Parse(proxyURL); err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxy),
			}
		}
	}

	return &Client{
		httpClient: client,
		proxy:      proxyURL,
	}
}

// DownloadFile downloads a file from URL to destination
func (c *Client) DownloadFile(url, dest string) error {
	resp, err := c.httpClient.Get(url)
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
	resp, err := c.httpClient.Get(url)
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
	resp, err := c.httpClient.Get(url)
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
	resp, err := c.httpClient.Get(url)
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
