package crawler

import (
	"sync"

	"censei/api"
	"censei/filter"
	"censei/logging"
	"censei/output"
)

// CallbackFunc is a function signature for worker callbacks
type CallbackFunc func()

// Worker coordinates parallel crawling of hosts
type Worker struct {
	client              *Client
	parser              *Parser
	filter              *filter.Filter
	writer              *output.Writer
	logger              *logging.Logger
	maxWorkers          int
	onHostOnline        CallbackFunc
	onFileFound         CallbackFunc
	onFilteredFileFound CallbackFunc
}

// NewWorker creates a new worker for coordinating crawling
func NewWorker(
	client *Client,
	parser *Parser,
	filter *filter.Filter,
	writer *output.Writer,
	logger *logging.Logger,
	maxWorkers int,
) *Worker {
	return &Worker{
		client:              client,
		parser:              parser,
		filter:              filter,
		writer:              writer,
		logger:              logger,
		maxWorkers:          maxWorkers,
		onHostOnline:        func() {}, // Default empty callbacks
		onFileFound:         func() {},
		onFilteredFileFound: func() {},
	}
}

// SetCallbacks sets the callback functions for the worker
func (w *Worker) SetCallbacks(
	onHostOnline CallbackFunc,
	onFileFound CallbackFunc,
	onFilteredFileFound CallbackFunc,
) {
	if onHostOnline != nil {
		w.onHostOnline = onHostOnline
	}
	if onFileFound != nil {
		w.onFileFound = onFileFound
	}
	if onFilteredFileFound != nil {
		w.onFilteredFileFound = onFilteredFileFound
	}
}

// ProcessHosts crawls each host in parallel
func (w *Worker) ProcessHosts(hosts []api.Host) {
	w.logger.Info("Starting to process %d hosts", len(hosts))

	// Create a channel for hosts
	hostChan := make(chan api.Host, len(hosts))

	// Fill the channel with hosts
	for _, host := range hosts {
		hostChan <- host
	}
	close(hostChan)

	// Create a wait group to track workers
	var wg sync.WaitGroup

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
	w.logger.Info("Processing host: %s", host.URL)

	// Check if host is online
	online, err := w.client.CheckHostOnline(host)
	if err != nil {
		w.logger.Error("Error checking host %s: %v", host.URL, err)
		return
	}

	if !online {
		w.logger.Info("Host is offline: %s", host.URL)
		return
	}

	// Host is online, trigger callback
	w.onHostOnline()

	// Host is online, write to output
	w.writer.WriteRawOutput(host.URL)

	// Fetch directory index
	htmlContent, err := w.client.FetchDirectoryIndex(host)
	if err != nil {
		w.logger.Error("Error fetching index from %s: %v", host.URL, err)
		return
	}

	// Parse directory index
	files, err := w.parser.ParseDirectoryIndex(host.URL, htmlContent)
	if err != nil {
		w.logger.Error("Error parsing index from %s: %v", host.URL, err)
		return
	}

	// Process found files
	for _, file := range files {
		// Trigger callback for file found
		w.onFileFound()

		// Write to raw output
		w.writer.WriteRawOutput("Found file: " + file.URL)

		// Apply filters
		if w.filter.ShouldFilter(file.URL) {
			w.logger.Info("File matched filter: %s", file.URL)
			// Trigger callback for filtered file
			w.onFilteredFileFound()
			// Write to filtered output
			w.writer.WriteFilteredOutput(file.URL)
		}
	}
}
