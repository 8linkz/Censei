package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"censei/config"
	"censei/logging"
)

// CensysClient handles interactions with the Censys CLI
type CensysClient struct {
	APIID     string
	APISecret string
	Config    *config.Config
	Logger    *logging.Logger
}

// NewCensysClient creates a new client for Censys API interactions
func NewCensysClient(apiID, apiSecret string, cfg *config.Config, logger *logging.Logger) *CensysClient {
	return &CensysClient{
		APIID:     apiID,
		APISecret: apiSecret,
		Config:    cfg,
		Logger:    logger,
	}
}

// ExecuteQuery runs a Censys search query and saves results to a JSON file
func (c *CensysClient) ExecuteQuery(query, outputDir string) (string, error) {
	// Create output filename
	outputPath := filepath.Join(outputDir, "censys_results.json")

	c.Logger.Info("Executing Censys query: %s", query)
	c.Logger.Debug("Output will be saved to: %s", outputPath)

	// Build command with config values
	c.Logger.Debug("Creating censys command with API credentials and config parameters")
	cmd := exec.Command(
		"censys", "search",
		"--api-id", c.APIID,
		"--api-secret", c.APISecret,
		"--page", strconv.Itoa(c.Config.LegacyPages),
		"--per-page", strconv.Itoa(c.Config.LegacyPerPage),
		"--index-type", c.Config.LegacyIndexType,
		"--sort-order", c.Config.LegacySortOrder,
		"--virtual-hosts", c.Config.LegacyVirtualHosts,
		"--output", outputPath,
		query,
	)

	// Create a buffer to capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	c.Logger.Debug("Executing censys command...")
	err := cmd.Run()
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	c.Logger.Debug("Command completed with stdout: %s", stdoutStr)
	if stderrStr != "" {
		c.Logger.Debug("Command stderr: %s", stderrStr)
	}

	if err != nil {
		c.Logger.Error("Censys command failed: %v", err)
		return "", fmt.Errorf("censys CLI error: %s: %w", stderrStr, err)
	}

	c.Logger.Info("Censys query completed successfully")

	// Verify the output file exists
	c.Logger.Debug("Checking if output file exists: %s", outputPath)
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		c.Logger.Error("Output file does not exist: %s", outputPath)
		return "", fmt.Errorf("censys did not create output file")
	}

	// Check file content
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		c.Logger.Error("Error getting file info: %v", err)
	} else {
		c.Logger.Debug("Output file size: %d bytes", fileInfo.Size())
	}

	return outputPath, nil
}

// ExtractHostsFromResults processes Censys JSON results and extracts hosts for crawling
func (c *CensysClient) ExtractHostsFromResults(jsonPath string) ([]Host, error) {
	c.Logger.Info("Extracting hosts from Censys results")

	// Read the JSON file
	c.Logger.Debug("Reading JSON file: %s", jsonPath)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		c.Logger.Error("Failed to read results file: %v", err)
		return nil, fmt.Errorf("failed to read results file: %w", err)
	}

	c.Logger.Debug("Read %d bytes from JSON file", len(data))

	// Log the first 500 characters for debugging
	if len(data) > 0 {
		previewLength := 500
		if len(data) < previewLength {
			previewLength = len(data)
		}
		c.Logger.Debug("JSON preview: %s", string(data[:previewLength]))
	} else {
		c.Logger.Debug("JSON file is empty")
		return nil, fmt.Errorf("JSON file is empty")
	}

	// Parse the JSON
	var results []CensysResult
	c.Logger.Debug("Attempting to parse JSON as array")
	err = json.Unmarshal(data, &results)
	if err != nil {
		c.Logger.Debug("Failed to parse JSON as array, trying alternative format: %v", err)

		// It might be an object with a results array
		var wrapper struct {
			Results []CensysResult `json:"results"`
		}

		c.Logger.Debug("Attempting to parse JSON as wrapper object")
		err = json.Unmarshal(data, &wrapper)
		if err != nil {
			c.Logger.Error("Failed to parse results JSON in any format: %v", err)
			return nil, fmt.Errorf("failed to parse results JSON in any format: %w", err)
		}

		results = wrapper.Results
		c.Logger.Debug("Successfully parsed JSON as wrapper object with %d results", len(results))
	} else {
		c.Logger.Debug("Successfully parsed JSON as array with %d results", len(results))
	}

	// Extract hosts - pre-allocate with estimated capacity
	// Estimate: results × average services per result (typically 2-5)
	estimatedHosts := len(results) * 3
	hosts := make([]Host, 0, estimatedHosts)

	for i, result := range results {
		c.Logger.Debug("Processing result #%d: IP=%s, Services=%d",
			i, result.IP, len(result.Services))

		// Determine base address (hostname)
		baseAddress := result.IP
		if len(result.DNS.ReverseDNS.Names) > 0 {
			baseAddress = result.DNS.ReverseDNS.Names[0]
			c.Logger.Debug("Using DNS name for host: %s", baseAddress)
		}

		// Use matched_services if available, otherwise fall back to services
		servicesToProcess := result.Services
		if len(result.MatchedServices) > 0 {
			c.Logger.Debug("Using %d matched services for host instead of all services",
				len(result.MatchedServices))
			servicesToProcess = result.MatchedServices
		}

		// Extract each HTTP service
		for j, service := range servicesToProcess {
			// Only process HTTP services
			if service.ServiceName != "HTTP" && service.ServiceName != "HTTPS" {
				continue
			}

			protocol := "http"
			if service.ServiceName == "HTTPS" || service.Port == 443 {
				protocol = "https"
			}

			// Format address for URL (add brackets for IPv6)
			addressForURL := baseAddress
			if isIPv6(baseAddress) {
				addressForURL = fmt.Sprintf("[%s]", baseAddress)
			}

			host := Host{
				BaseAddress: baseAddress,
				IP:          result.IP,
				Port:        service.Port,
				Protocol:    protocol,
				URL:         fmt.Sprintf("%s://%s:%d", protocol, addressForURL, service.Port),
			}

			// Special case for standard ports
			switch service.Port {
			case 443:
				host.URL = fmt.Sprintf("https://%s", addressForURL)
			case 80:
				host.URL = fmt.Sprintf("http://%s", addressForURL)
			}

			c.Logger.Debug("Created host #%d.%d: %s", i, j, host.URL)
			hosts = append(hosts, host)
		}
	}

	c.Logger.Debug("Extracted %d hosts from Censys results", len(hosts))
	return hosts, nil
}
