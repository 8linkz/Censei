package api

// CensysResult represents a result item from Censys API
type CensysResult struct {
	IP              string    `json:"ip"`
	DNS             DNS       `json:"dns"`
	Services        []Service `json:"services"`
	MatchedServices []Service `json:"matched_services"`
}

// DNS contains DNS information from Censys
type DNS struct {
	ReverseDNS ReverseDNS `json:"reverse_dns"`
}

// ReverseDNS contains reverse DNS information
type ReverseDNS struct {
	Names []string `json:"names"`
}

// Service represents a service from a Censys result
type Service struct {
	Port                int    `json:"port"`
	ServiceName         string `json:"service_name"`
	ExtendedServiceName string `json:"extended_service_name"`
	TransportProtocol   string `json:"transport_protocol"`
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
