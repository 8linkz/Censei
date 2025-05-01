package api

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"censei/logging"
)

// CensysClient handles interactions with the Censys CLI
type CensysClient struct {
	APIID     string
	APISecret string
	Logger    *logging.Logger
}

// NewCensysClient creates a new client for Censys API interactions
func NewCensysClient(apiID, apiSecret string, logger *logging.Logger) *CensysClient {
	return &CensysClient{
		APIID:     apiID,
		APISecret: apiSecret,
		Logger:    logger,
	}
}

// ExecuteQuery runs a Censys search query and saves results to a JSON file
func (c *CensysClient) ExecuteQuery(query, outputDir string) (string, error) {
	// Create output filename
	outputPath := filepath.Join(outputDir, "censys_results.json")

	c.Logger.Info("Executing Censys query: %s", query)
	c.Logger.Debug("Output will be saved to: %s", outputPath)

	// Build command
	cmd := exec.Command(
		"censys", "search",
		"--api-id", c.APIID,
		"--api-secret", c.APISecret,
		"--page", "25",
		"--per-page", "100",
		"--index-type", "hosts",
		"--sort-order", "DESCENDING",
		"--virtual-hosts", "INCLUDE",
		"--output", outputPath,
		query,
	)

	// Run command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("censys CLI error: %s: %w", string(output), err)
	}

	c.Logger.Info("Censys query completed successfully")
	c.Logger.Debug("Censys CLI output: %s", string(output))

	// Verify the output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("censys did not create output file")
	}

	return outputPath, nil
}

// ExtractHostsFromResults processes Censys JSON results and extracts hosts for crawling
func (c *CensysClient) ExtractHostsFromResults(jsonPath string) ([]Host, error) {
	c.Logger.Info("Extracting hosts from Censys results")

	// Read the JSON file
	data, err := os.ReadFile(jsonPath)
	if err != nil {
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
	err = json.Unmarshal(data, &results)
	if err != nil {
		c.Logger.Debug("Failed to parse JSON as array, trying alternative format: %v", err)

		// It might be an object with a results array
		var wrapper struct {
			Results []CensysResult `json:"results"`
		}

		err = json.Unmarshal(data, &wrapper)
		if err != nil {
			return nil, fmt.Errorf("failed to parse results JSON in any format: %w", err)
		}

		results = wrapper.Results
	}

	c.Logger.Debug("Parsed %d results from JSON", len(results))

	// Extract hosts
	hosts := make([]Host, 0)

	for i, result := range results {
		c.Logger.Debug("Processing result #%d: IP=%s, Services=%d",
			i, result.IP, len(result.Services))

		// Determine base address (hostname)
		baseAddress := result.IP
		if len(result.DNS.ReverseDNS.Names) > 0 {
			baseAddress = result.DNS.ReverseDNS.Names[0]
			c.Logger.Debug("Using DNS name for host: %s", baseAddress)
		}

		// Extract each HTTP service
		for j, service := range result.Services {
			// Only process HTTP services
			if service.ServiceName != "HTTP" && service.ServiceName != "HTTPS" {
				continue
			}

			protocol := "http"
			if service.ServiceName == "HTTPS" || service.Port == 443 {
				protocol = "https"
			}

			host := Host{
				BaseAddress: baseAddress,
				IP:          result.IP,
				Port:        service.Port,
				Protocol:    protocol,
				URL:         fmt.Sprintf("%s://%s:%d", protocol, baseAddress, service.Port),
			}

			// Special case for standard ports
			if service.Port == 443 {
				host.URL = fmt.Sprintf("https://%s", baseAddress)
			} else if service.Port == 80 {
				host.URL = fmt.Sprintf("http://%s", baseAddress)
			}

			c.Logger.Debug("Created host #%d.%d: %s", i, j, host.URL)
			hosts = append(hosts, host)
		}
	}

	c.Logger.Info("Extracted %d hosts from Censys results", len(hosts))
	return hosts, nil
}
