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

	// Parse the JSON
	var results []CensysResult
	err = json.Unmarshal(data, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to parse results JSON: %w", err)
	}

	// Extract hosts
	hosts := make([]Host, 0)

	for _, result := range results {
		// Use name if available, otherwise use IP
		baseAddress := result.Name
		if baseAddress == "" {
			baseAddress = result.IP
		}

		// Extract each service
		for _, service := range result.MatchedServices {
			protocol := "http"
			if service.Port == 443 {
				protocol = "https"
			}

			host := Host{
				BaseAddress: baseAddress,
				IP:          result.IP,
				Port:        service.Port,
				Protocol:    protocol,
				URL:         fmt.Sprintf("%s://%s:%d", protocol, baseAddress, service.Port),
			}

			// Special case for port 443 (https)
			if service.Port == 443 {
				host.URL = fmt.Sprintf("https://%s", baseAddress)
			} else if service.Port == 80 {
				host.URL = fmt.Sprintf("http://%s", baseAddress)
			}

			hosts = append(hosts, host)
		}
	}

	c.Logger.Info("Extracted %d hosts from Censys results", len(hosts))
	return hosts, nil
}
