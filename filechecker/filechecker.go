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

// NewFileChecker creates a new file checker instance with optimized connection pooling
func NewFileChecker(timeoutSeconds int, logger *logging.Logger) *FileChecker {
	// Create a custom transport with relaxed TLS settings
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			// MinVersion is set to allow connections to older servers
			// Note: This is a security risk but needed for scanning purposes
		},
		MaxIdleConns:          200,               // Increased from 100
		MaxIdleConnsPerHost:   20,                // Increased from 10 (was default 2)
		MaxConnsPerHost:       0,                 // No limit on total connections
		IdleConnTimeout:       90 * time.Second,  // Keep connections alive longer
		DisableCompression:    true,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     false,             // Disable HTTP/2 for simpler scanning
		MaxResponseHeaderBytes: 10 << 20,         // 10 MB max header size
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

// isBinaryContentType checks if a content type indicates binary content
// Optimized helper to avoid code duplication and enable early exit
func isBinaryContentType(contentType string) bool {
	binaryTypes := []string{
		// Generic binary types
		"application/octet-stream",
		"application/binary",

		// Windows executable types
		"application/x-executable",
		"application/x-msdos-program",
		"application/x-msdownload",
		"application/exe",
		"application/x-dosexec",
		"application/vnd.microsoft.portable-executable",
		"application/x-ms-dos-executable",

		// Windows library and installer types
		"application/x-dll",
		"application/x-msdownload", // DLL variant
		"application/x-msi",
		"application/x-ms-installer",
		"application/vnd.ms-msi",
		"application/vnd.microsoft.portable-executable", // DLL/EXE
		"application/vnd.ms-cab-compressed",
		"application/x-ms-shortcut", // .lnk files
		"application/x-ms-screensaver", // .scr files
		"application/x-com", // .com files

		// Linux executable and package types
		"application/x-elf",
		"application/x-sharedlib",
		"application/x-executable",
		"application/vnd.debian.binary-package",
		"application/x-deb",
		"application/x-debian-package",
		"application/x-rpm",
		"application/x-redhat-package-manager",
		"application/x-iso9660-appimage",
		"application/vnd.appimage",

		// macOS executable and package types
		"application/x-apple-diskimage",
		"application/x-newton-compatible-pkg",
		"application/x-mac-package",
		"application/vnd.apple.installer+xml",
		"application/x-mach-binary",
		"application/x-mach-object",
		"application/x-apple-bundle",

		// Android/Mobile package types
		"application/vnd.android.package-archive", // .apk files

		// Archive types (ZIP)
		"application/zip",
		"application/x-zip",
		"application/x-zip-compressed",
		"application/x-compress",
		"application/x-compressed",
		"multipart/x-zip",

		// Archive types (RAR, 7Z, TAR, GZ)
		"application/x-rar",
		"application/x-rar-compressed",
		"application/vnd.rar",
		"application/x-7z-compressed",
		"application/x-tar",
		"application/x-gzip",
		"application/gzip",

		// Archive types (BZ2, XZ, ISO)
		"application/x-bzip2",
		"application/x-bzip",
		"application/x-xz",
		"application/x-lzma",
		"application/x-iso9660-image",
		"application/x-cd-image",

		// Script types (potentially malicious)
		"application/x-sh",
		"application/x-shellscript",
		"application/x-bash",
		"application/x-bat",
		"application/x-msdos-batch",
		"application/x-vbscript",
		"text/vbscript",
		"application/x-javascript", // Standalone .js files
		"application/javascript", // Can be malicious
		"application/java-archive",
		"application/x-java-archive",
		"application/x-jar",
		"application/x-powershell",
		"application/x-ms-powershell",
	}

	for _, binaryType := range binaryTypes {
		if strings.Contains(contentType, binaryType) {
			return true // Early exit on first match
		}
	}
	return false
}

// CheckSpecificFile checks if a specific file exists at the given URL
// and verifies its content type without downloading the full file
func (fc *FileChecker) CheckSpecificFile(baseURL, fileName string) (bool, string, error) {
	if !fc.checkEnabled {
		return false, "", fmt.Errorf("file checking functionality is disabled")
	}

	// Validate fileName to prevent path traversal attacks
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return false, "", fmt.Errorf("invalid file name: contains path traversal characters")
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

	// Check content length
	contentLength := resp.ContentLength
	if contentLength == 0 {
		return false, contentType, fmt.Errorf("file has zero size")
	}

	// Check for binary content types using optimized helper
	isBinaryContent := isBinaryContentType(contentType)

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

	fc.logger.Debug("Checking file: %s", fileURL)

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
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("server returned non-OK status: %d", resp.StatusCode)
	}

	// Get content type
	contentType := resp.Header.Get("Content-Type")

	// Check content length
	contentLength := resp.ContentLength
	if contentLength == 0 {
		return false, contentType, fmt.Errorf("file has zero size")
	}

	// Check for binary content types using optimized helper
	isBinaryContent := isBinaryContentType(contentType)

	// Log the result
	if isBinaryContent {
		fc.logger.Info("Found binary file at %s with Content-Type: %s", fileURL, contentType)
		return true, contentType, nil
	}

	fc.logger.Debug("File found but not binary content: %s (Content-Type: %s)",
		fileURL, contentType)
	return false, contentType, fmt.Errorf("file is not binary content")
}
