package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// LogLine mirrors the LogEntry schema used by the parser.
type LogLine struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	SourceIP  string `json:"source_ip"`
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
}

func main() {
	const totalLines = 10_000
	const outputFile = "test_logs/sample_logs.json"

	levels := []string{"INFO", "WARN", "ERROR"}

	messages := []string{
		"User login successful",
		"Failed login attempt",
		"Session token refreshed",
		"Password reset requested",
		"Unauthorized API access detected",
		"New device registered",
		"Rate limit exceeded",
		"Account locked after repeated failures",
		"Firewall rule triggered",
		"Suspicious IP geo-hop detected",
	}

	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	baseTime := time.Now().UTC()

	for i := 0; i < totalLines; i++ {
		entry := LogLine{
			Timestamp: baseTime.Add(time.Duration(i) * time.Millisecond).Format(time.RFC3339Nano),
			Level:     levels[rand.Intn(len(levels))],
			SourceIP:  fmt.Sprintf("192.168.%d.%d", rand.Intn(256), rand.Intn(256)),
			UserID:    fmt.Sprintf("user_%04d", rand.Intn(500)),
			Message:   messages[rand.Intn(len(messages))],
		}

		if err := encoder.Encode(entry); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing entry %d: %v\n", i, err)
			os.Exit(1)
		}
	}

	// Seed attacker IPs to trigger brute-force detection
	attackerIPs := []string{"10.0.0.66", "10.0.0.77", "10.0.0.88"}
	for _, ip := range attackerIPs {
		for j := 0; j < 20; j++ {
			entry := LogLine{
				Timestamp: baseTime.Add(time.Duration(totalLines+j) * time.Millisecond).Format(time.RFC3339Nano),
				Level:     "WARN",
				SourceIP:  ip,
				UserID:    "user_0001",
				Message:   "Failed login attempt",
			}
			if err := encoder.Encode(entry); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing attacker entry: %v\n", err)
				os.Exit(1)
			}
		}
	}

	fmt.Printf("Generated %d log entries + %d attacker entries → %s\n", totalLines, len(attackerIPs)*20, outputFile)
}
