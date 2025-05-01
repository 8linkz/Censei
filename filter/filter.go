package filter

import (
	"path/filepath"
	"strings"

	"censei/logging"
)

// Filter handles file filtering based on extensions
type Filter struct {
	extensions []string
	logger     *logging.Logger
}

// NewFilter creates a new filter with the given extensions
func NewFilter(extensions []string, logger *logging.Logger) *Filter {
	// Ensure all extensions start with a dot
	for i, ext := range extensions {
		if !strings.HasPrefix(ext, ".") {
			extensions[i] = "." + ext
		}
	}

	return &Filter{
		extensions: extensions,
		logger:     logger,
	}
}

// ShouldFilter checks if a file should be filtered based on its extension
func (f *Filter) ShouldFilter(fileURL string) bool {
	// No filters defined
	if len(f.extensions) == 0 {
		return false
	}

	// Get the file extension
	ext := strings.ToLower(filepath.Ext(fileURL))

	// Check if the extension is in our filter list
	for _, filterExt := range f.extensions {
		if strings.ToLower(filterExt) == ext {
			f.logger.Debug("File %s matches filter %s", fileURL, filterExt)
			return true
		}
	}

	return false
}

// GetFilterExtensions returns the current filter extensions
func (f *Filter) GetFilterExtensions() []string {
	return f.extensions
}
