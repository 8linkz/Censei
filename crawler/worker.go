package crawler

import (
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"

	"censei/api"
	"censei/config"
	"censei/filechecker"
	"censei/filter"
	"censei/logging"
	"censei/output"
	"censei/scanners"
)

// Worker coordinates parallel crawling of hosts
type Worker struct {
	client           *Client
	filter           *filter.Filter
	writer           *output.Writer
	logger           *logging.Logger
	fileChecker      *filechecker.FileChecker
	directoryScanner *scanners.DirectoryScanner
	queryConfig      *config.Query
	config           *config.Config
	maxWorkers       int
	checkEnabled     bool
	targetFileName   string
	foundUrls        *sync.Map // Map for deduplication
	skippedHosts     *sync.Map
	blockedHosts     *sync.Map
	skipCounters     *sync.Map
	stats            *ScanStats
}

// ScanStats tracks statistics during scanning
type ScanStats struct {
	totalHosts       int
	onlineHosts      int
	totalFiles       int
	filteredFiles    int
	checkedFiles     int
	binaryFilesFound int
	mu               sync.Mutex
}

// NewWorker creates a new worker for coordinating crawling
func NewWorker(
	client *Client,
	filter *filter.Filter,
	writer *output.Writer,
	logger *logging.Logger,
	queryConfig *config.Query,
	config *config.Config,
	maxWorkers int,
) *Worker {
	return &Worker{
		client:           client,
		filter:           filter,
		writer:           writer,
		logger:           logger,
		directoryScanner: scanners.NewDirectoryScanner(logger),
		queryConfig:      queryConfig,
		config:           config,
		maxWorkers:       maxWorkers,
		foundUrls:        &sync.Map{},
		skippedHosts:     &sync.Map{},
		stats:            &ScanStats{},
	}
}

// SetFileChecker configures the file checker for the worker
func (w *Worker) SetFileChecker(checker *filechecker.FileChecker, enabled bool, targetFileName string) {
	w.fileChecker = checker
	w.checkEnabled = enabled
	w.targetFileName = targetFileName

	// Configure file checker if present
	if w.fileChecker != nil {
		w.fileChecker.Configure(enabled, targetFileName)
	}
}

// ProcessHosts crawls each host in parallel
func (w *Worker) ProcessHosts(hosts []api.Host) {
	w.logger.Info("Starting to process %d hosts", len(hosts))
	w.stats.totalHosts = len(hosts)

	// Create channels for parallel processing
	hostChan := make(chan api.Host, len(hosts))
	var wg sync.WaitGroup

	// Fill the channel with hosts
	for _, host := range hosts {
		hostChan <- host
	}
	close(hostChan)

	// Start workers
	for i := 0; i < w.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for host := range hostChan {
				w.processHost(host)
			}
		}()
	}

	// Wait for all workers to finish
	wg.Wait()
	w.logger.Info("Finished processing all hosts")
}

// processHost handles a single host's crawling and scanning
func (w *Worker) processHost(host api.Host) {
	// Log the host we're processing
	w.logger.Debug("Processing host: %s", host.URL)

	// Extract base host for blocking checks
	baseHost := w.extractBaseHost(host.URL)

	// Check if entire base host is blocked
	if _, isBlocked := w.blockedHosts.Load(baseHost); isBlocked {
		w.logger.Info("Skipping host - base host is blocked: %s", host.URL)
		return
	}

	// Check if this host should be skipped due to limits
	if _, shouldSkip := w.skippedHosts.Load(host.URL); shouldSkip {
		w.logger.Info("Skipping host due to previous limit exceeded: %s", host.URL)
		return
	}

	// Check if host is online and fetch content
	online, htmlContent, err := w.client.CheckHostAndFetch(host)
	if err != nil {
		w.logger.Error("Error checking host %s: %v", host.URL, err)
		return
	}

	if !online {
		w.logger.Debug("Host is offline: %s", host.URL)
		return
	}

	// Update stats for online host
	w.stats.mu.Lock()
	w.stats.onlineHosts++
	w.stats.mu.Unlock()

	// Host is online, write to output
	w.writer.WriteRawOutput(host.URL)

	// Check if this is a targeted check mode
	targetedCheckMode := w.checkEnabled && w.fileChecker != nil && w.targetFileName != ""

	// Try to check for a specific file if configured
	if targetedCheckMode {
		w.logger.Info("Checking for specific file %s at %s", w.targetFileName, host.URL)

		found, contentType, err := w.fileChecker.CheckSpecificFile(host.URL, w.targetFileName)
		if err == nil && found {
			w.logger.Info("Found binary file '%s' at %s with Content-Type: %s",
				w.targetFileName, host.URL, contentType)
			binaryOutput := fmt.Sprintf("%s/%s with Content-Type: %s",
				host.URL, w.targetFileName, contentType)
			w.writer.WriteRawOutput(fmt.Sprintf("Found binary file: %s", binaryOutput))
			w.writer.WriteBinaryOutput(binaryOutput)

			// Update check statistics
			w.stats.mu.Lock()
			w.stats.checkedFiles++
			w.stats.binaryFilesFound++
			w.stats.mu.Unlock()
		} else if err != nil {
			w.logger.Debug("Failed to check for specific file: %v", err)
		}
	}

	// Process directory content if not in targeted mode or if no file found yet
	if !targetedCheckMode || w.stats.binaryFilesFound == 0 {
		w.processDirectoryContent(host, htmlContent)
	}
}

// processDirectoryContent handles directory listing scanning and file processing
func (w *Worker) processDirectoryContent(host api.Host, htmlContent string) {
	// Check if content is a directory listing
	if !w.directoryScanner.IsDirectoryListing(htmlContent) {
		w.logger.Debug("Host content is not a directory listing: %s", host.URL)
		return
	}

	var fileURLs []string

	// Check if recursive scanning is enabled
	recursive := w.queryConfig.Recursive == "yes"
	maxDepth := w.queryConfig.MaxDepth

	// Create skip callback function with block logic
	skipCallback := func(hostURL string) {
		baseHost := w.extractBaseHost(hostURL)
		w.logger.Info("Marking host for skip due to link limit: %s", hostURL)
		w.skippedHosts.Store(host.URL, true) // Mark the original host, not subdirectory

		// Increment skip counter for base host
		skipCount, _ := w.skipCounters.LoadOrStore(baseHost, int64(0))
		newSkipCount := atomic.AddInt64(skipCount.(*int64), 1)

		w.logger.Debug("Skip count for base host %s: %d", baseHost, newSkipCount)

		// Check if we should block entire base host
		if w.config.MaxSkipsBeforeBlock > 0 && newSkipCount >= int64(w.config.MaxSkipsBeforeBlock) {
			w.logger.Info("Blocking entire base host after %d skips: %s", newSkipCount, baseHost)
			w.blockedHosts.Store(baseHost, true)
		}
	}

	if recursive && maxDepth > 1 {
		w.logger.Info("Starting recursive scan with max-depth %d for %s", maxDepth, host.URL)
		fileURLs = w.directoryScanner.ScanHostRecursive(host, htmlContent, maxDepth, w.client, w.config, skipCallback)
	} else {
		w.logger.Debug("Using normal directory scan for %s", host.URL)
		fileURLs = w.directoryScanner.ScanHost(host, htmlContent)
	}

	// Process each found file
	for _, fileURL := range fileURLs {
		w.processFoundFile(fileURL)
	}
}

// processFoundFile handles individual file processing including filtering and checking
func (w *Worker) processFoundFile(fileURL string) {
	// Check if we've already found this URL
	if _, exists := w.foundUrls.LoadOrStore(fileURL, true); exists {
		w.logger.Debug("Skipping duplicate URL: %s", fileURL)
		return
	}

	// Update stats for file found
	w.stats.mu.Lock()
	w.stats.totalFiles++
	w.stats.mu.Unlock()

	// Write to raw output
	w.writer.WriteRawOutput("Found file: " + fileURL)

	// Apply filters
	if w.filter.ShouldFilter(fileURL) {
		w.logger.Info("File matched filter: %s", fileURL)

		// Update stats for filtered file
		w.stats.mu.Lock()
		w.stats.filteredFiles++
		w.stats.mu.Unlock()

		// Write to filtered output
		w.writer.WriteFilteredOutput(fileURL)

		// Check file content type if enabled
		if w.checkEnabled && w.fileChecker != nil && w.fileChecker.ShouldCheck(fileURL) {
			w.checkFileContent(fileURL)
		}
	}
}

// checkFileContent verifies if a file contains binary content
func (w *Worker) checkFileContent(fileURL string) {
	found, contentType, err := w.fileChecker.CheckFileURL(fileURL)
	if err == nil && found {
		w.logger.Info("Found binary file at %s with Content-Type: %s", fileURL, contentType)
		w.writer.WriteRawOutput(fmt.Sprintf("Found binary file: %s", fileURL))
		w.writer.WriteBinaryOutput(fileURL)

		// Update check statistics
		w.stats.mu.Lock()
		w.stats.checkedFiles++
		w.stats.binaryFilesFound++
		w.stats.mu.Unlock()
	} else if err != nil {
		w.logger.Debug("File check failed for %s: %v", fileURL, err)
	}

	// Update checked files statistic
	w.stats.mu.Lock()
	w.stats.checkedFiles++
	w.stats.mu.Unlock()
}

// GetStats returns the current scan statistics
func (w *Worker) GetStats() (int, int, int, int, int, int) {
	return w.stats.totalHosts, w.stats.onlineHosts, w.stats.totalFiles,
		w.stats.filteredFiles, w.stats.checkedFiles, w.stats.binaryFilesFound
}

// extractBaseHost extracts the base host (IP:port or domain:port) from a full URL
func (w *Worker) extractBaseHost(fullURL string) string {
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		w.logger.Debug("Failed to parse URL %s: %v", fullURL, err)
		return fullURL // Fallback to full URL if parsing fails
	}

	// Return scheme + host (includes port if present)
	return parsedURL.Scheme + "://" + parsedURL.Host
}
