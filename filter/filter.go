package filter

import (
	"path/filepath"
	"strings"

	"censei/logging"
)

// Filter handles file filtering based on extensions with O(1) map lookup
type Filter struct {
	extensionMap map[string]bool
	logger       *logging.Logger
}

// NewFilter creates a new filter with the given extensions
// Extensions are normalized once during initialization for optimal performance
func NewFilter(extensions []string, logger *logging.Logger) *Filter {
	// Create map for O(1) lookup instead of O(n) slice iteration
	extensionMap := make(map[string]bool, len(extensions))

	for _, ext := range extensions {
		// Ensure extension starts with a dot
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		// Store as lowercase for case-insensitive matching
		// Normalization happens once here instead of on every file check
		extensionMap[strings.ToLower(ext)] = true
	}

	return &Filter{
		extensionMap: extensionMap,
		logger:       logger,
	}
}

// ShouldFilter checks if a file should be filtered based on its extension
// Uses O(1) map lookup for optimal performance
func (f *Filter) ShouldFilter(fileURL string) bool {
	// No filters defined
	if len(f.extensionMap) == 0 {
		return false
	}

	// Get the file extension and convert to lowercase once
	ext := strings.ToLower(filepath.Ext(fileURL))

	// O(1) map lookup instead of O(n) loop
	if f.extensionMap[ext] {
		f.logger.Debug("File %s matches filter extension %s", fileURL, ext)
		return true
	}

	return false
}

// GetFilterExtensions returns the current filter extensions as a slice
func (f *Filter) GetFilterExtensions() []string {
	// Convert map keys back to slice for compatibility
	extensions := make([]string, 0, len(f.extensionMap))
	for ext := range f.extensionMap {
		extensions = append(extensions, ext)
	}
	return extensions
}
