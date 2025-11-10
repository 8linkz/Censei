package output

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"censei/logging"
)

// BinaryFinding represents a binary file finding with its URL and Content-Type
type BinaryFinding struct {
	URL         string
	ContentType string
}

// Writer handles output file operations with buffered I/O for performance
type Writer struct {
	rawFile      *os.File
	filteredFile *os.File
	binaryFile   *os.File
	rawWriter      *bufio.Writer
	filteredWriter *bufio.Writer
	binaryWriter   *bufio.Writer
	mu           sync.Mutex
	logger       *logging.Logger

	// Collect binary findings grouped by host for sorted output
	binaryFindings map[string][]BinaryFinding // host -> list of findings
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

	// Create buffered writers for 10-100x faster writes
	// Default buffer size: 4096 bytes (bufio.defaultBufSize)
	// For high-throughput scanning, use 64KB buffers
	const bufferSize = 64 * 1024 // 64 KB

	return &Writer{
		rawFile:        rawFile,
		filteredFile:   filteredFile,
		binaryFile:     binaryFile,
		rawWriter:      bufio.NewWriterSize(rawFile, bufferSize),
		filteredWriter: bufio.NewWriterSize(filteredFile, bufferSize),
		binaryWriter:   bufio.NewWriterSize(binaryFile, bufferSize),
		logger:         logger,
		binaryFindings: make(map[string][]BinaryFinding),
	}, nil
}

// WriteRawOutput writes a line to the raw output file using buffered I/O
func (w *Writer) WriteRawOutput(line string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, err := fmt.Fprintln(w.rawWriter, line)
	if err != nil {
		w.logger.Error("Failed to write to raw output: %v", err)
		return err
	}

	return nil
}

// WriteFilteredOutput writes a line to the filtered output file using buffered I/O
func (w *Writer) WriteFilteredOutput(line string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, err := fmt.Fprintln(w.filteredWriter, line)
	if err != nil {
		w.logger.Error("Failed to write to filtered output: %v", err)
		return err
	}

	return nil
}

// WriteBinaryOutput collects binary findings grouped by host for sorted output
// Expected line format: "URL with Content-Type: CONTENT_TYPE"
func (w *Writer) WriteBinaryOutput(line string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Parse the line to extract URL and Content-Type
	// Format: "http://example.com/file.exe with Content-Type: application/x-msdownload"
	parts := strings.Split(line, " with Content-Type: ")
	if len(parts) != 2 {
		w.logger.Error("Invalid binary output format: %s", line)
		return fmt.Errorf("invalid binary output format")
	}

	fileURL := strings.TrimSpace(parts[0])
	contentType := strings.TrimSpace(parts[1])

	// Extract host from URL
	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		w.logger.Error("Failed to parse URL %s: %v", fileURL, err)
		return err
	}

	host := parsedURL.Scheme + "://" + parsedURL.Host

	// Check if this URL already exists for this host to avoid duplicates
	for _, existing := range w.binaryFindings[host] {
		if existing.URL == fileURL {
			// URL already recorded, skip duplicate
			return nil
		}
	}

	// Add finding to the map
	w.binaryFindings[host] = append(w.binaryFindings[host], BinaryFinding{
		URL:         fileURL,
		ContentType: contentType,
	})

	return nil
}

// writeSortedBinaryFindings writes all binary findings grouped by host in sorted order
func (w *Writer) writeSortedBinaryFindings() error {
	if len(w.binaryFindings) == 0 {
		return nil
	}

	// Sort hosts alphabetically
	hosts := make([]string, 0, len(w.binaryFindings))
	for host := range w.binaryFindings {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)

	// Write findings grouped by host
	for _, host := range hosts {
		findings := w.binaryFindings[host]
		if len(findings) == 0 {
			continue
		}

		// Write host separator
		separator := fmt.Sprintf("\n=== %s (%d files) ===\n", host, len(findings))
		if _, err := w.binaryWriter.WriteString(separator); err != nil {
			return fmt.Errorf("failed to write host separator: %w", err)
		}

		// Write all findings for this host (URLs only for easy copying)
		for _, finding := range findings {
			line := fmt.Sprintf("%s\n", finding.URL)
			if _, err := w.binaryWriter.WriteString(line); err != nil {
				return fmt.Errorf("failed to write binary finding: %w", err)
			}
		}
	}

	return nil
}

// Close flushes buffers and closes all output files
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.logger.Info("Closing output files and flushing buffers")

	var rawFlushErr, filteredFlushErr, binaryFlushErr error
	var rawErr, filteredErr, binaryErr error

	// Flush all buffers first to ensure data is written
	if w.rawWriter != nil {
		rawFlushErr = w.rawWriter.Flush()
		if rawFlushErr != nil {
			w.logger.Error("Failed to flush raw output buffer: %v", rawFlushErr)
		}
		w.rawWriter = nil
	}

	if w.filteredWriter != nil {
		filteredFlushErr = w.filteredWriter.Flush()
		if filteredFlushErr != nil {
			w.logger.Error("Failed to flush filtered output buffer: %v", filteredFlushErr)
		}
		w.filteredWriter = nil
	}

	// Write sorted binary findings before flushing
	if w.binaryWriter != nil {
		w.logger.Info("Writing %d binary findings grouped by host", len(w.binaryFindings))
		binaryFlushErr = w.writeSortedBinaryFindings()
		if binaryFlushErr != nil {
			w.logger.Error("Failed to write sorted binary findings: %v", binaryFlushErr)
		}

		// Now flush the buffer
		flushErr := w.binaryWriter.Flush()
		if flushErr != nil {
			w.logger.Error("Failed to flush binary output buffer: %v", flushErr)
			if binaryFlushErr == nil {
				binaryFlushErr = flushErr
			}
		}
		w.binaryWriter = nil
	}

	// Close files after flushing
	if w.rawFile != nil {
		rawErr = w.rawFile.Close()
		if rawErr != nil {
			w.logger.Error("Failed to close raw output file: %v", rawErr)
		}
		w.rawFile = nil
	}

	if w.filteredFile != nil {
		filteredErr = w.filteredFile.Close()
		if filteredErr != nil {
			w.logger.Error("Failed to close filtered output file: %v", filteredErr)
		}
		w.filteredFile = nil
	}

	if w.binaryFile != nil {
		binaryErr = w.binaryFile.Close()
		if binaryErr != nil {
			w.logger.Error("Failed to close binary output file: %v", binaryErr)
		}
		w.binaryFile = nil
	}

	// Return first error encountered
	if rawFlushErr != nil {
		return rawFlushErr
	}
	if filteredFlushErr != nil {
		return filteredFlushErr
	}
	if binaryFlushErr != nil {
		return binaryFlushErr
	}
	if rawErr != nil {
		return rawErr
	}
	if filteredErr != nil {
		return filteredErr
	}
	if binaryErr != nil {
		return binaryErr
	}

	w.logger.Info("Output files closed successfully")
	return nil
}
