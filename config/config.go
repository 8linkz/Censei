package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds application configuration
type Config struct {
	APIKey                string `json:"api_key"`
	APISecret             string `json:"api_secret"`
	OutputDir             string `json:"output_dir"`
	HTTPTimeoutSeconds    int    `json:"http_timeout_seconds"`
	MaxConcurrentRequests int    `json:"max_concurrent_requests"`
	LogLevel              string `json:"log_level"`
	LogFile               string `json:"log_file"`
}

// Query represents a predefined Censys query with its filters
type Query struct {
	Name    string   `json:"name"`
	Query   string   `json:"query"`
	Filters []string `json:"filters"`
}

// LoadConfig loads and validates the application configuration from a file
func LoadConfig(path string) (*Config, error) {
	// Set default values
	config := &Config{
		OutputDir:             "./output",
		HTTPTimeoutSeconds:    5,
		MaxConcurrentRequests: 10,
		LogLevel:              "INFO",
		LogFile:               "./censei.log",
	}

	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate config
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadQueries loads predefined queries from a file
func LoadQueries(path string) ([]Query, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read queries file: %w", err)
	}

	var queries []Query
	err = json.Unmarshal(data, &queries)
	if err != nil {
		return nil, fmt.Errorf("failed to parse queries file: %w", err)
	}

	return queries, nil
}

// validateConfig ensures that required fields are present
func validateConfig(cfg *Config) error {
	if cfg.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}
	if cfg.APISecret == "" {
		return fmt.Errorf("api_secret is required")
	}
	if cfg.HTTPTimeoutSeconds <= 0 {
		cfg.HTTPTimeoutSeconds = 5 // Default to 5 seconds
	}
	if cfg.MaxConcurrentRequests <= 0 {
		cfg.MaxConcurrentRequests = 10 // Default to 10 concurrent requests
	}

	// Create output directory if it doesn't exist
	if _, err := os.Stat(cfg.OutputDir); os.IsNotExist(err) {
		err := os.MkdirAll(cfg.OutputDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	return nil
}
