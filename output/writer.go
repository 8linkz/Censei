package output

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"censei/logging"
)

// Writer handles output file operations
type Writer struct {
	rawFile      *os.File
	filteredFile *os.File
	binaryFile   *os.File
	mu           sync.Mutex
	logger       *logging.Logger
	seenUrls     map[string]bool // Track already seen URLs
}

// NewWriter creates a new output writer
func NewWriter(outputDir string, logger *logging.Logger) (*Writer, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create raw output file
	rawPath := filepath.Join(outputDir, "raw.txt")
	rawFile, err := os.Create(rawPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create raw output file: %w", err)
	}

	// Create filtered output file
	filteredPath := filepath.Join(outputDir, "filtered.txt")
	filteredFile, err := os.Create(filteredPath)
	if err != nil {
		rawFile.Close()
		return nil, fmt.Errorf("failed to create filtered output file: %w", err)
	}

	// Create binary output file
	binaryPath := filepath.Join(outputDir, "binary_found.txt")
	binaryFile, err := os.Create(binaryPath)
	if err != nil {
		rawFile.Close()
		filteredFile.Close()
		return nil, fmt.Errorf("failed to create binary output file: %w", err)
	}

	logger.Info("Output files created: %s, %s and %s", rawPath, filteredPath, binaryPath)

	return &Writer{
		rawFile:      rawFile,
		filteredFile: filteredFile,
		binaryFile:   binaryFile,
		logger:       logger,
		seenUrls:     make(map[string]bool), // Initialize the map here
	}, nil
}

// WriteRawOutput writes a line to the raw output file
func (w *Writer) WriteRawOutput(line string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if we've already written this line
	if w.seenUrls[line] {
		w.logger.Debug("Skipping duplicate raw output: %s", line)
		return nil
	}

	// Mark as seen
	w.seenUrls[line] = true

	_, err := fmt.Fprintln(w.rawFile, line)
	if err != nil {
		w.logger.Error("Failed to write to raw output: %v", err)
		return err
	}

	return nil
}

// WriteFilteredOutput writes a line to the filtered output file
func (w *Writer) WriteFilteredOutput(line string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// For filtered output, we use a different key to track uniqueness
	filterKey := "filtered:" + line

	// Check if we've already written this line to filtered output
	if w.seenUrls[filterKey] {
		w.logger.Debug("Skipping duplicate filtered output: %s", line)
		return nil
	}

	// Mark as seen
	w.seenUrls[filterKey] = true

	_, err := fmt.Fprintln(w.filteredFile, line)
	if err != nil {
		w.logger.Error("Failed to write to filtered output: %v", err)
		return err
	}

	return nil
}

// WriteBinaryOutput writes a line to the binary output file
func (w *Writer) WriteBinaryOutput(line string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// For binary output, we use a different key to track uniqueness
	binaryKey := "binary:" + line

	// Check if we've already written this line to binary output
	if w.seenUrls[binaryKey] {
		w.logger.Debug("Skipping duplicate binary output: %s", line)
		return nil
	}

	// Mark as seen
	w.seenUrls[binaryKey] = true

	_, err := fmt.Fprintln(w.binaryFile, line)
	if err != nil {
		w.logger.Error("Failed to write to binary output: %v", err)
		return err
	}

	return nil
}

// Close closes all output files
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var rawErr, filteredErr, binaryErr error

	if w.rawFile != nil {
		rawErr = w.rawFile.Close()
		w.rawFile = nil
	}

	if w.filteredFile != nil {
		filteredErr = w.filteredFile.Close()
		w.filteredFile = nil
	}

	if w.binaryFile != nil {
		binaryErr = w.binaryFile.Close()
		w.binaryFile = nil
	}

	if rawErr != nil {
		return rawErr
	}

	if filteredErr != nil {
		return filteredErr
	}

	return binaryErr
}
