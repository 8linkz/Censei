package api

// CensysResult represents a result item from Censys API
type CensysResult struct {
	IP              string           `json:"ip"`
	Name            string           `json:"name"`
	MatchedServices []MatchedService `json:"matched_services"`
}

// MatchedService represents a service from a Censys result
type MatchedService struct {
	Port int `json:"port"`
}

// Host represents a processed host for crawling
type Host struct {
	BaseAddress string
	IP          string
	Port        int
	Protocol    string
	URL         string
}

// FoundFile represents a file found during crawling
type FoundFile struct {
	URL          string
	HostURL      string
	RelativePath string
	Filtered     bool
}
