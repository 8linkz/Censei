package crawler

import (
	"context"
	"crypto/tls"
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
	// Create a custom transport with optimized settings
	transport := &http.Transport{
		MaxIdleConns:          100,
		IdleConnTimeout:       30 * time.Second,
		ResponseHeaderTimeout: time.Duration(timeoutSeconds) * time.Second,
		DisableKeepAlives:     true, // Prevent "unsolicited response" messages
		DisableCompression:    true, // Speed up response processing
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Skip SSL certificate verification
		},
	}

	client := &http.Client{
		Timeout:   time.Duration(timeoutSeconds) * time.Second,
		Transport: transport,
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

// CheckHostAndFetch combines checking if host is online and fetching its content
// Returns if the host is online, the HTML content (if any), and any error
func (c *Client) CheckHostAndFetch(host api.Host) (bool, string, error) {
	c.logger.Debug("Checking host and fetching content: %s", host.URL)

	ctx, cancel := context.WithTimeout(context.Background(), c.httpClient.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", host.URL, nil)
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to avoid blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CenseiBot/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Debug("Host offline or unreachable: %s (%s)", host.URL, err)
		return false, "", nil // Not an error, just offline
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		c.logger.Debug("Host responded with non-OK status: %s (Status: %d)", host.URL, resp.StatusCode)
		return false, "", nil
	}

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return true, "", fmt.Errorf("failed to read response body: %w", err)
	}

	c.logger.Debug("Host online: %s (Status: %d, Content length: %d bytes)",
		host.URL, resp.StatusCode, len(bodyBytes))

	return true, string(bodyBytes), nil
}
