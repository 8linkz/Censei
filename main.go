package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"censei/api"
	"censei/cli"
	"censei/config"
	"censei/crawler"
	"censei/filter"
	"censei/logging"
	"censei/output"
)

// checkCensysCLI checks if the censys-cli tool is available
func checkCensysCLI(logger *logging.Logger) bool {
	logger.Info("Checking if censys-cli is installed...")

	cmd := exec.Command("censys", "--version")
	err := cmd.Run()

	if err != nil {
		logger.Error("censys-cli is not available or not in PATH: %v", err)
		fmt.Println("ERROR: The censys-cli tool was not found. Please install it with:")
		fmt.Println("pip install censys-command-line")
		fmt.Println("or visit https://github.com/censys/censys-command-line for installation instructions.")
		return false
	}

	logger.Info("censys-cli is installed and available")
	return true
}

func main() {
	// Parse command line arguments
	configPath := flag.String("config", "./config.json", "Path to config file")
	queriesPath := flag.String("queries", "./queries.json", "Path to queries file")
	filterStr := flag.String("filter", "", "Custom file extensions to filter (comma-separated, e.g. .pdf,.exe)")
	queryStr := flag.String("query", "", "Run specific query directly")
	outputPath := flag.String("output", "", "Override output directory")
	logLevel := flag.String("log-level", "", "Override log level (DEBUG, INFO, ERROR)")
	flag.Parse()

	// Initialize logging system
	logger := logging.NewLogger()

	// Check if censys-cli is installed
	if !checkCensysCLI(logger) {
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Error("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Override config with command line arguments if provided
	if *outputPath != "" {
		cfg.OutputDir = *outputPath
	}
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}

	// Apply log level from config
	logger.SetLevel(cfg.LogLevel)
	logger.SetOutputFile(cfg.LogFile)

	// Initialize the application
	logger.Info("Censei Scanner starting up...")

	// Load queries configuration
	queries, err := config.LoadQueries(*queriesPath)
	if err != nil {
		logger.Error("Failed to load queries: %v", err)
		os.Exit(1)
	}

	// If a direct query is provided, run it
	if *queryStr != "" {
		logger.Info("Running direct query: %s", *queryStr)
		// Create a filter slice from the filter string
		var filters []string
		if *filterStr != "" {
			filters = cli.ParseFilters(*filterStr)
		}
		// Run the query
		runQuery(cfg, *queryStr, filters, logger)
	} else {
		// Start interactive mode
		selectedQuery, selectedFilters := cli.ShowMenu(queries, *filterStr)
		if selectedQuery == "" {
			logger.Error("No query selected, exiting")
			os.Exit(0)
		}
		runQuery(cfg, selectedQuery, selectedFilters, logger)
	}
}

func runQuery(cfg *config.Config, query string, filters []string, logger *logging.Logger) {
	startTime := time.Now()

	// Initialize Censys client
	censysClient := api.NewCensysClient(cfg.APIKey, cfg.APISecret, logger)

	// Execute Censys query
	jsonPath, err := censysClient.ExecuteQuery(query, cfg.OutputDir)
	if err != nil {
		logger.Error("Failed to execute Censys query: %v", err)
		os.Exit(1)
	}

	// Extract hosts from results
	hosts, err := censysClient.ExtractHostsFromResults(jsonPath)
	if err != nil {
		logger.Error("Failed to extract hosts from results: %v", err)
		os.Exit(1)
	}

	logger.Info("Extracted %d hosts from Censys results", len(hosts))

	// Initialize output writer
	writer, err := output.NewWriter(cfg.OutputDir, logger)
	if err != nil {
		logger.Error("Failed to initialize output writer: %v", err)
		os.Exit(1)
	}
	defer writer.Close()

	// Initialize filter
	fileFilter := filter.NewFilter(filters, logger)
	logger.Info("Using filters: %v", fileFilter.GetFilterExtensions())

	// Initialize client
	client := crawler.NewClient(cfg.HTTPTimeoutSeconds, logger)

	// Initialize the optimized worker
	worker := crawler.NewWorker(
		client,
		fileFilter,
		writer,
		logger,
		cfg.MaxConcurrentRequests,
	)

	// Process hosts
	worker.ProcessHosts(hosts)

	// Get statistics
	totalHosts, onlineHosts, totalFiles, filteredFiles := worker.GetStats()

	// Generate and write summary
	endTime := time.Now()
	summary := output.FormatSummary(
		query,
		totalHosts,
		onlineHosts,
		totalFiles,
		filteredFiles,
		fileFilter.GetFilterExtensions(),
		startTime,
		endTime,
	)

	logger.Info("\n%s", summary)
	writer.WriteRawOutput("\n" + summary)

	logger.Info("Query execution complete")
}
