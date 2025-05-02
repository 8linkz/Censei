package filechecker

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"censei/logging"
)

// FileChecker handles file verification operations without downloading
type FileChecker struct {
	httpClient     *http.Client
	logger         *logging.Logger
	checkEnabled   bool
	targetFileName string
}

// NewFileChecker creates a new file checker instance
func NewFileChecker(timeoutSeconds int, logger *logging.Logger) *FileChecker {
	// Create a custom transport with relaxed TLS settings
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			// MinVersion is set to allow connections to older servers
			// Note: This is a security risk but needed for scanning purposes
		},
		MaxIdleConns:       100,
		IdleConnTimeout:    5 * time.Second,
		DisableCompression: true,
	}

	client := &http.Client{
		Timeout:   time.Duration(timeoutSeconds) * time.Second,
		Transport: transport,
	}

	return &FileChecker{
		httpClient:     client,
		logger:         logger,
		checkEnabled:   false,
		targetFileName: "",
	}
}

// Configure sets up the file checker options
func (fc *FileChecker) Configure(enabled bool, targetFileName string) {
	fc.checkEnabled = enabled
	fc.targetFileName = targetFileName
}

// CheckSpecificFile checks if a specific file exists at the given URL
// and verifies its content type without downloading the full file
func (fc *FileChecker) CheckSpecificFile(baseURL, fileName string) (bool, string, error) {
	if !fc.checkEnabled {
		return false, "", fmt.Errorf("file checking functionality is disabled")
	}

	// Clean up the base URL
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Construct full URL
	fileURL := fmt.Sprintf("%s/%s", baseURL, fileName)
	fc.logger.Info("Checking for specific file: %s", fileURL)

	// Create the request
	req, err := http.NewRequest("GET", fileURL, nil)
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to avoid detection/blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CenseiBot/1.0)")
	req.Header.Set("Accept", "*/*")

	// Execute the request
	resp, err := fc.httpClient.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to check file: %w", err)
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("server returned non-OK status: %d", resp.StatusCode)
	}

	// Get content type
	contentType := resp.Header.Get("Content-Type")

	// Check for binary content types
	isBinaryContent := strings.Contains(contentType, "application/octet-stream") ||
		strings.Contains(contentType, "application/x-executable") ||
		strings.Contains(contentType, "application/x-msdos-program") ||
		strings.Contains(contentType, "application/x-msdownload") ||
		strings.Contains(contentType, "application/exe") ||
		strings.Contains(contentType, "application/binary")

	// Read a small portion of the body to verify content type
	// This helps avoid downloading the entire file
	buffer := make([]byte, 512)
	n, err := io.ReadAtLeast(resp.Body, buffer, 1)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		n = 0
	}

	// Log the result
	if isBinaryContent {
		fc.logger.Info("Found '%s' at %s with Content-Type: %s", fileName, fileURL, contentType)
		return true, contentType, nil
	}

	fc.logger.Debug("File found but not binary content: %s (Content-Type: %s, First bytes: %x)",
		fileURL, contentType, buffer[:n])
	return false, contentType, fmt.Errorf("file is not binary content")
}

// ShouldCheck determines if a file should be checked
func (fc *FileChecker) ShouldCheck(fileURL string) bool {
	// If check is not enabled, don't check anything
	if !fc.checkEnabled {
		return false
	}

	// If a target filename is specified, check if this URL matches
	if fc.targetFileName != "" {
		baseName := filepath.Base(fileURL)
		return baseName == fc.targetFileName
	}

	// If no target filename, check all files
	return true
}

// CheckFileURL checks if a file at the given URL is binary content
func (fc *FileChecker) CheckFileURL(fileURL string) (bool, string, error) {
	if !fc.checkEnabled {
		return false, "", fmt.Errorf("file checking functionality is disabled")
	}

	fc.logger.Info("Checking file: %s", fileURL)

	// Create the request
	req, err := http.NewRequest("HEAD", fileURL, nil)
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CenseiBot/1.0)")
	req.Header.Set("Accept", "*/*")

	// Execute HEAD request first to check content type efficiently
	resp, err := fc.httpClient.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to check file: %w", err)
	}
	resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("server returned non-OK status: %d", resp.StatusCode)
	}

	// Get content type
	contentType := resp.Header.Get("Content-Type")

	// Check for binary content types
	isBinaryContent := strings.Contains(contentType, "application/octet-stream") ||
		strings.Contains(contentType, "application/x-executable") ||
		strings.Contains(contentType, "application/x-msdos-program") ||
		strings.Contains(contentType, "application/x-msdownload") ||
		strings.Contains(contentType, "application/exe") ||
		strings.Contains(contentType, "application/binary")

	// Log the result
	if isBinaryContent {
		fc.logger.Info("Found binary file at %s with Content-Type: %s", fileURL, contentType)
		return true, contentType, nil
	}

	fc.logger.Debug("File found but not binary content: %s (Content-Type: %s)",
		fileURL, contentType)
	return false, contentType, fmt.Errorf("file is not binary content")
}
