package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"censei/config"
)

// ShowMenu displays an interactive menu for query selection
func ShowMenu(queries []config.Query, customFilterStr string) (string, []string) {
	// Display menu options
	fmt.Println("\nSelect query:")
	for i, q := range queries {
		fmt.Printf("%d. %s\n", i+1, q.Name)
	}
	fmt.Printf("%d. Custom query\n", len(queries)+1)

	// Read user input
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nEnter selection number: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

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

			return query, filters
		}

		// Predefined query
		selectedQuery := queries[num-1]

		// Use custom filters if provided, otherwise use the predefined ones
		filters := selectedQuery.Filters
		if customFilterStr != "" {
			filters = ParseFilters(customFilterStr)
		}

		return selectedQuery.Query, filters
	}
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
