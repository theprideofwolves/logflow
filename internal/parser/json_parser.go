package parser

import (
	"encoding/json"
	"fmt"
	"time"
)

// LogEntry represents a single structured log event from the input stream.
// Fields are chosen to support security-oriented pattern detection.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	SourceIP  string    `json:"source_ip"`
	UserID    string    `json:"user_id"`
}

// ParseLine takes a single raw JSON log line and converts it into a LogEntry.
// It returns a structured entry on success, or a descriptive error if the
// input is malformed or missing required fields.
func ParseLine(data []byte) (LogEntry, error) {
	if len(data) == 0 {
		return LogEntry{}, fmt.Errorf("parse error: input is empty")
	}

	var entry LogEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return LogEntry{}, fmt.Errorf("parse error: invalid JSON: %w", err)
	}

	// Validate that critical fields are present
	if entry.Timestamp.IsZero() {
		return LogEntry{}, fmt.Errorf("parse error: missing or invalid 'timestamp' field")
	}
	if entry.Level == "" {
		return LogEntry{}, fmt.Errorf("parse error: missing 'level' field")
	}
	if entry.Message == "" {
		return LogEntry{}, fmt.Errorf("parse error: missing 'message' field")
	}

	return entry, nil
}
