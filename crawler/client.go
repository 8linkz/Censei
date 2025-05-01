package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"censei/api"
	"censei/logging"
)

// Client handles HTTP requests for crawling
type Client struct {
	httpClient *http.Client
	logger     *logging.Logger
}

// NewClient creates a new crawler client
func NewClient(timeoutSeconds int, logger *logging.Logger) *Client {
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects
			return http.ErrUseLastResponse
		},
	}

	return &Client{
		httpClient: client,
		logger:     logger,
	}
}

// CheckHostOnline verifies if a host is online by sending a HEAD request
func (c *Client) CheckHostOnline(host api.Host) (bool, error) {
	c.logger.Debug("Checking if host is online: %s", host.URL)

	ctx, cancel := context.WithTimeout(context.Background(), c.httpClient.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", host.URL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to avoid blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CenseiBot/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Debug("Host offline or unreachable: %s (%s)", host.URL, err)
		return false, nil // Not an error, just offline
	}
	defer resp.Body.Close()

	// Check if status code is 200 OK
	if resp.StatusCode == http.StatusOK {
		c.logger.Debug("Host online: %s (Status: %d)", host.URL, resp.StatusCode)
		return true, nil
	}

	c.logger.Debug("Host responded with non-OK status: %s (Status: %d)", host.URL, resp.StatusCode)
	return false, nil
}

// FetchDirectoryIndex retrieves the directory index page
func (c *Client) FetchDirectoryIndex(host api.Host) (string, error) {
	c.logger.Debug("Fetching directory index for: %s", host.URL)

	ctx, cancel := context.WithTimeout(context.Background(), c.httpClient.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", host.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to avoid blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CenseiBot/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch index: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned non-OK status: %d", resp.StatusCode)
	}

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(bodyBytes), nil
}
