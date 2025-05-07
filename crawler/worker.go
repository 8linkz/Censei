package crawler

import (
	"fmt"
	"sync"

	"censei/api"
	"censei/filechecker"
	"censei/filter"
	"censei/logging"
	"censei/output"
)

// Worker coordinates parallel crawling of hosts
type Worker struct {
	client         *Client
	filter         *filter.Filter
	writer         *output.Writer
	logger         *logging.Logger
	fileChecker    *filechecker.FileChecker
	maxWorkers     int
	checkEnabled   bool
	targetFileName string
	foundUrls      *sync.Map // Map for deduplication
	stats          *ScanStats
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
	maxWorkers int,
) *Worker {
	return &Worker{
		client:     client,
		filter:     filter,
		writer:     writer,
		logger:     logger,
		maxWorkers: maxWorkers,
		foundUrls:  &sync.Map{},
		stats:      &ScanStats{},
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

// processHost handles a single host's crawling
func (w *Worker) processHost(host api.Host) {
	// Log the host we're processing
	w.logger.Debug("Processing host: %s", host.URL)

	// Check if host is online
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
		w.logger.Info("Checking for specific file %s at %s",
			w.targetFileName, host.URL)

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

	// Parse directory index and process files
	// If in targeted mode and we only want to check for a specific file, we may skip directory scanning
	if !(w.checkEnabled && w.targetFileName != "") {
		// Parse directory index and process files directly
		links := extractLinks(host.URL, htmlContent, w.logger)

		for _, fileURL := range links {
			// Check if we've already found this URL
			if _, exists := w.foundUrls.LoadOrStore(fileURL, true); exists {
				w.logger.Debug("Skipping duplicate URL: %s", fileURL)
				continue
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
					found, contentType, err := w.fileChecker.CheckFileURL(fileURL)
					if err == nil && found {
						w.logger.Info("Found binary file at %s with Content-Type: %s", fileURL, contentType)
						binaryOutput := fmt.Sprintf("%s with Content-Type: %s", fileURL, contentType)
						w.writer.WriteRawOutput(fmt.Sprintf("Found binary file: %s", binaryOutput))
						w.writer.WriteBinaryOutput(binaryOutput)

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
			}
		}
	}
}

// GetStats returns the current scan statistics
func (w *Worker) GetStats() (int, int, int, int, int, int) {
	return w.stats.totalHosts, w.stats.onlineHosts, w.stats.totalFiles,
		w.stats.filteredFiles, w.stats.checkedFiles, w.stats.binaryFilesFound
}
