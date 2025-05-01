package cli

import (
	"strings"
)

// ParseCommandLineArgs parses command line arguments for non-interactive mode
func ParseCommandLineArgs(args []string) map[string]string {
	result := make(map[string]string)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check if the argument is a flag
		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")

			// If there's a value following the flag
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				result[key] = args[i+1]
				i++ // Skip the next argument since we've consumed it
			} else {
				// Flag without a value
				result[key] = "true"
			}
		}
	}

	return result
}

// BuildCensysCommand constructs a command to execute the Censys CLI
func BuildCensysCommand(apiID, apiSecret, query string) []string {
	// Base command
	cmd := []string{
		"censys", "search",
		"--api-id", apiID,
		"--api-secret", apiSecret,
		"--page", "25",
		"--per-page", "100",
		"--index-type", "hosts",
		"--sort-order", "DESCENDING",
		"--virtual-hosts", "INCLUDE",
	}

	// Add the query
	cmd = append(cmd, query)

	return cmd
}
