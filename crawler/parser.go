package crawler

import (
	"net/url"
	"strings"

	"censei/api"
	"censei/logging"

	"github.com/PuerkitoBio/goquery"
)

// Parser handles HTML parsing for directory indices
type Parser struct {
	logger *logging.Logger
}

// NewParser creates a new HTML parser
func NewParser(logger *logging.Logger) *Parser {
	return &Parser{
		logger: logger,
	}
}

// ParseDirectoryIndex extracts links from a directory index HTML page
func (p *Parser) ParseDirectoryIndex(hostURL, htmlContent string) ([]api.FoundFile, error) {
	p.logger.Debug("Parsing directory index for: %s", hostURL)

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(hostURL)
	if err != nil {
		return nil, err
	}

	var files []api.FoundFile

	// Find all links
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// Skip parent directory links
		if href == "../" || href == ".." || href == "." || href == "/" {
			return
		}

		// Resolve relative URLs
		fileURL, err := url.Parse(href)
		if err != nil {
			p.logger.Debug("Failed to parse URL: %s", href)
			return
		}

		absoluteURL := baseURL.ResolveReference(fileURL).String()

		// Create found file
		file := api.FoundFile{
			URL:          absoluteURL,
			HostURL:      hostURL,
			RelativePath: href,
			Filtered:     false,
		}

		files = append(files, file)
		p.logger.Debug("Found file: %s", absoluteURL)
	})

	p.logger.Info("Found %d files in directory index for %s", len(files), hostURL)
	return files, nil
}
