package patterns

import (
	"fmt"
	"sync"

	"github.com/varkuru/logflow/internal/parser"
)

// ThreatDetector tracks suspicious activity across log entries.
// It is safe for concurrent use by multiple worker goroutines.
type ThreatDetector struct {
	mu           sync.Mutex
	failuresByIP map[string]int
	alertedIPs   map[string]bool
	Threshold    int
	TotalThreats int
}

// NewThreatDetector creates a detector with the given failure threshold.
func NewThreatDetector(threshold int) *ThreatDetector {
	return &ThreatDetector{
		failuresByIP: make(map[string]int),
		alertedIPs:   make(map[string]bool),
		Threshold:    threshold,
	}
}

// Analyze inspects a single LogEntry for brute-force indicators.
// If an IP exceeds the failure threshold, it prints a threat warning.
func (td *ThreatDetector) Analyze(entry parser.LogEntry) {
	// Only count entries that indicate a failed action
	if !isFailure(entry) || entry.SourceIP == "" {
		return
	}

	td.mu.Lock()
	defer td.mu.Unlock()

	ip := entry.SourceIP
	td.failuresByIP[ip]++

	if td.failuresByIP[ip] > td.Threshold && !td.alertedIPs[ip] {
		fmt.Printf("  ⚠ THREAT DETECTED: Brute force from IP %s (%d failures)\n", ip, td.failuresByIP[ip])
		td.alertedIPs[ip] = true
		td.TotalThreats++
	}
}

// Summary returns the total number of unique IPs that triggered alerts.
func (td *ThreatDetector) Summary() (totalIPs int, totalThreats int) {
	td.mu.Lock()
	defer td.mu.Unlock()
	return len(td.failuresByIP), td.TotalThreats
}

// isFailure checks if a log entry represents a failed action based on
// its message content and log level.
func isFailure(entry parser.LogEntry) bool {
	switch entry.Message {
	case "Failed login attempt",
		"Unauthorized API access detected",
		"Rate limit exceeded",
		"Account locked after repeated failures":
		return true
	}
	if entry.Level == "ERROR" {
		return true
	}
	return false
}
