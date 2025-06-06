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
	"censei/filechecker"
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
	checkFlag := flag.Bool("check", false, "Enable targeted file checking mode - skips HTML processing and link extraction, directly checks hosts for specific binary files")
	targetFile := flag.String("target-file", "", "Specific file to check for on hosts")
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
		// Run the query with check options
		runQuery(cfg, *queryStr, filters, *checkFlag, *targetFile, logger)
	} else {
		// Start interactive mode
		selectedQuery, selectedFilters, checkEnabled, targetFileName := cli.ShowMenuWithCheck(
			queries, *filterStr, *checkFlag, *targetFile)
		if selectedQuery == "" {
			logger.Error("No query selected, exiting")
			os.Exit(0)
		}
		runQuery(cfg, selectedQuery, selectedFilters, checkEnabled, targetFileName, logger)
	}
}

func runQuery(cfg *config.Config, query string, filters []string, check bool, targetFileName string, logger *logging.Logger) {
	startTime := time.Now()

	// Initialize statistics
	stats := struct {
		totalHosts       int
		onlineHosts      int
		totalFiles       int
		filteredFiles    int
		checkedFiles     int
		binaryFilesFound int
	}{
		totalHosts:       0,
		onlineHosts:      0,
		totalFiles:       0,
		filteredFiles:    0,
		checkedFiles:     0,
		binaryFilesFound: 0,
	}

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

	// Initialize crawler components
	client := crawler.NewClient(cfg.HTTPTimeoutSeconds, logger)

	// Initialize worker
	worker := crawler.NewWorker(
		client,
		fileFilter,
		writer,
		logger,
		cfg.MaxConcurrentRequests,
	)

	// Initialize file checker if enabled
	if check {
		logger.Info("File checking functionality enabled, looking for binary files")
		if targetFileName != "" {
			logger.Info("Target filename: %s", targetFileName)
		}

		// Create file checker
		fileChecker := filechecker.NewFileChecker(cfg.HTTPTimeoutSeconds, logger)

		// Set file checker in worker
		worker.SetFileChecker(fileChecker, true, targetFileName)
	}

	// Process hosts
	worker.ProcessHosts(hosts)

	// Get updated statistics
	stats.totalHosts, stats.onlineHosts, stats.totalFiles, stats.filteredFiles, stats.checkedFiles, stats.binaryFilesFound = worker.GetStats()

	// Generate and write summary
	endTime := time.Now()
	summary := output.FormatSummary(
		query,
		stats.totalHosts,
		stats.onlineHosts,
		stats.totalFiles,
		stats.filteredFiles,
		stats.checkedFiles,
		stats.binaryFilesFound,
		fileFilter.GetFilterExtensions(),
		startTime,
		endTime,
		check,
		targetFileName,
		cfg.BinaryOutputFile,
	)

	logger.Info("\n%s", summary)
	writer.WriteRawOutput("\n" + summary)

	logger.Info("Query execution complete")
}
