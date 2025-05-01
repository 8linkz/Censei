package crawler

import (
	"net/url"
	"strings"

	"censei/logging"

	"github.com/PuerkitoBio/goquery"
)

// extractLinks extracts links from HTML content without creating a parser object
func extractLinks(baseURLStr string, htmlContent string, logger *logging.Logger) []string {
	var links []string

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		logger.Error("Failed to parse HTML: %v", err)
		return links
	}

	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		logger.Error("Failed to parse base URL: %v", err)
		return links
	}

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
			logger.Debug("Failed to parse URL: %s", href)
			return
		}

		absoluteURL := baseURL.ResolveReference(fileURL).String()
		links = append(links, absoluteURL)
		logger.Debug("Found link: %s", absoluteURL)
	})

	logger.Info("Found %d links in directory index for %s", len(links), baseURLStr)
	return links
}
