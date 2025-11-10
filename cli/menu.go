package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"censei/config"
)

// ShowMenuWithCheck displays an interactive menu for query selection with file checking options
func ShowMenuWithCheck(queries []config.Query, customFilterStr string, defaultCheck bool, defaultTargetFile string, isLegacyMode bool) (string, []string, bool, string) {
	// Display banner with mode indication
	PrintBannerWithMode(isLegacyMode)

	reader := bufio.NewReader(os.Stdin)

	// Pagination settings
	const queriesPerPage = 25
	currentPage := 0
	totalPages := (len(queries) + queriesPerPage - 1) / queriesPerPage

	for {
		// Display current page
		start := currentPage * queriesPerPage
		end := start + queriesPerPage
		if end > len(queries) {
			end = len(queries)
		}

		fmt.Println("\n═══════════════════════════════════════════════════════════════")
		if totalPages > 1 {
			fmt.Printf("Available Queries (Page %d/%d):\n", currentPage+1, totalPages)
		} else {
			fmt.Println("Available Queries:")
		}
		fmt.Println("═══════════════════════════════════════════════════════════════")

		// Display queries for current page
		for i := start; i < end; i++ {
			q := queries[i]
			// Build query info on one line
			line := fmt.Sprintf("[%d] %s", i+1, q.Name)

			// Add filters
			if len(q.Filters) > 0 {
				line += fmt.Sprintf(" - Filters: %s", strings.Join(q.Filters, ", "))
			}

			// Add recursive info
			if q.Recursive == "yes" {
				line += fmt.Sprintf(" | Recursive (depth: %d)", q.MaxDepth)
			}

			// Add target file info
			if q.Check && q.TargetFileName != "" {
				line += fmt.Sprintf(" | Target: %s", q.TargetFileName)
			}

			fmt.Println(line)
		}

		fmt.Println("\n───────────────────────────────────────────────────────────────")

		// Show navigation options
		if totalPages > 1 {
			fmt.Printf("\n[c] Custom query")
			if currentPage < totalPages-1 {
				fmt.Printf("  |  [n] Next page")
			}
			if currentPage > 0 {
				fmt.Printf("  |  [p] Previous page")
			}
			fmt.Println()
		} else {
			fmt.Println("\n[c] Custom query")
		}

		fmt.Println("\n═══════════════════════════════════════════════════════════════")
		fmt.Print("Enter selection: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		// Handle navigation commands
		if input == "n" && currentPage < totalPages-1 {
			currentPage++
			continue
		}
		if input == "p" && currentPage > 0 {
			currentPage--
			continue
		}
		if input == "c" {
			// Handle custom query (existing code below)
			input = strconv.Itoa(len(queries) + 1)
		}

		// Convert to number
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(queries)+1 {
			fmt.Println("Invalid selection. Please try again.")
			continue
		}

		// Custom query option
		if num == len(queries)+1 {
			fmt.Print("Enter custom Censys query: ")
			query, _ := reader.ReadString('\n')
			query = strings.TrimSpace(query)

			if query == "" {
				fmt.Println("Query cannot be empty. Please try again.")
				continue
			}

			// Parse custom filters if provided
			var filters []string
			if customFilterStr != "" {
				filters = ParseFilters(customFilterStr)
			} else {
				// Ask for filters
				fmt.Print("Enter filters (comma-separated file extensions, e.g. .pdf,.exe, or leave empty): ")
				filterInput, _ := reader.ReadString('\n')
				filterInput = strings.TrimSpace(filterInput)

				if filterInput != "" {
					filters = ParseFilters(filterInput)
				}
			}

			// Ask for file checking options
			check := defaultCheck
			if !defaultCheck {
				fmt.Print("Enable file content checking? (y/n): ")
				checkInput, _ := reader.ReadString('\n')
				checkInput = strings.TrimSpace(strings.ToLower(checkInput))
				check = checkInput == "y" || checkInput == "yes"
			}

			targetFile := defaultTargetFile
			if check && targetFile == "" {
				fmt.Print("Enter specific filename to check for (leave empty to check all filtered files): ")
				targetInput, _ := reader.ReadString('\n')
				targetFile = strings.TrimSpace(targetInput)
			}

			return query, filters, check, targetFile
		}

		// Predefined query
		selectedQuery := queries[num-1]

		// Use custom filters if provided, otherwise use the predefined ones
		filters := selectedQuery.Filters
		if customFilterStr != "" {
			filters = ParseFilters(customFilterStr)
		}

		// Use query's check settings, or override with command line if provided
		check := selectedQuery.Check
		if defaultCheck {
			check = true
		}

		targetFile := selectedQuery.TargetFileName
		if defaultTargetFile != "" {
			targetFile = defaultTargetFile
		}

		return selectedQuery.Query, filters, check, targetFile
	}
}

// ShowMenu provides backward compatibility with the original interface
func ShowMenu(queries []config.Query, customFilterStr string) (string, []string) {
	// Call the new function but only return the first two values (default to Platform API v3 mode)
	query, filters, _, _ := ShowMenuWithCheck(queries, customFilterStr, false, "", false)
	return query, filters
}

// ParseFilters converts a comma-separated string of file extensions to a slice
func ParseFilters(filterStr string) []string {
	filters := strings.Split(filterStr, ",")
	for i, filter := range filters {
		filters[i] = strings.TrimSpace(filter)
		// Ensure filter starts with a dot
		if !strings.HasPrefix(filters[i], ".") {
			filters[i] = "." + filters[i]
		}
	}
	return filters
}
