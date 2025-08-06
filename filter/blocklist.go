package filter

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"censei/logging"
)

// Blocklist manages a persistent list of blocked hosts
type Blocklist struct {
	hosts    map[string]time.Time // hostname -> timestamp when blocked
	filePath string
	enabled  bool
	logger   *logging.Logger
	mu       sync.RWMutex
}

// NewBlocklist creates a new blocklist instance
func NewBlocklist(filePath string, enabled bool, logger *logging.Logger) *Blocklist {
	return &Blocklist{
		hosts:    make(map[string]time.Time),
		filePath: filePath,
		enabled:  enabled,
		logger:   logger,
	}
}

// Load reads the blocklist from file if it exists
func (b *Blocklist) Load() error {
	if !b.enabled {
		b.logger.Debug("Blocklist is disabled, skipping load")
		return nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(b.filePath); os.IsNotExist(err) {
		b.logger.Info("Blocklist file does not exist, starting with empty blocklist: %s", b.filePath)
		return nil
	}

	// Open and read file
	file, err := os.Open(b.filePath)
	if err != nil {
		return fmt.Errorf("failed to open blocklist file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		// Parse line format: "hostname timestamp" or just "hostname"
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		hostname := parts[0]
		var timestamp time.Time

		if len(parts) >= 2 {
			// Try to parse timestamp
			if ts, err := time.Parse(time.RFC3339, parts[1]); err == nil {
				timestamp = ts
			} else {
				timestamp = time.Now() // Fallback to current time
			}
		} else {
			timestamp = time.Now()
		}

		b.hosts[hostname] = timestamp
		count++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading blocklist file: %w", err)
	}

	b.logger.Info("Loaded %d blocked hosts from %s", count, b.filePath)
	return nil
}

// Save writes the current blocklist to file
func (b *Blocklist) Save() error {
	if !b.enabled {
		return nil
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	file, err := os.Create(b.filePath)
	if err != nil {
		return fmt.Errorf("failed to create blocklist file: %w", err)
	}
	defer file.Close()

	// Write header comment
	fmt.Fprintf(file, "# Censei Blocklist - Generated on %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "# Format: hostname timestamp\n")
	fmt.Fprintf(file, "# Hosts that exceeded skip limits and are permanently blocked\n\n")

	// Write hosts
	for hostname, timestamp := range b.hosts {
		fmt.Fprintf(file, "%s %s\n", hostname, timestamp.Format(time.RFC3339))
	}

	b.logger.Info("Saved %d blocked hosts to %s", len(b.hosts), b.filePath)
	return nil
}

// IsBlocked checks if a host is in the blocklist
func (b *Blocklist) IsBlocked(hostname string) bool {
	if !b.enabled {
		return false
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	_, exists := b.hosts[hostname]
	return exists
}

// AddHost adds a host to the blocklist
func (b *Blocklist) AddHost(hostname string) {
	if !b.enabled {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.hosts[hostname]; !exists {
		b.hosts[hostname] = time.Now()
		b.logger.Info("Added host to blocklist: %s", hostname)

		// Auto-save after adding new host
		go func() {
			if err := b.Save(); err != nil {
				b.logger.Error("Failed to save blocklist after adding host: %v", err)
			}
		}()
	}
}

// GetBlockedCount returns the number of blocked hosts
func (b *Blocklist) GetBlockedCount() int {
	if !b.enabled {
		return 0
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.hosts)
}

// GetBlockedHosts returns a copy of all blocked hosts with timestamps
func (b *Blocklist) GetBlockedHosts() map[string]time.Time {
	if !b.enabled {
		return make(map[string]time.Time)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string]time.Time)
	for hostname, timestamp := range b.hosts {
		result[hostname] = timestamp
	}

	return result
}
