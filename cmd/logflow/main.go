package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/varkuru/logflow/internal/parser"
	"github.com/varkuru/logflow/internal/patterns"
	"github.com/varkuru/logflow/internal/worker"
)

func main() {
	const poolSize = 100
	const queueSize = 1000
	const threatThreshold = 5

	// --- Determine the input source ---
	var input *os.File
	var sourceName string

	if len(os.Args) > 1 {
		// File path provided as argument
		f, err := os.Open(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] Cannot open log file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		input = f
		sourceName = os.Args[1]
	} else {
		// No argument — read from stdin (piped input)
		input = os.Stdin
		sourceName = "stdin"
	}

	fmt.Printf("[ENGINE] Reading from: %s\n", sourceName)

	// --- Initialize the Threat Detector ---
	detector := patterns.NewThreatDetector(threatThreshold)

	// --- Define the processing callback for each worker ---
	processFn := func(payload []byte) {
		entry, err := parser.ParseLine(payload)
		if err != nil {
			return
		}
		detector.Analyze(entry)
	}

	// --- Start the Dispatcher with 100 workers ---
	dispatcher := worker.NewDispatcher(poolSize, queueSize, processFn)
	dispatcher.Start()
	fmt.Printf("[ENGINE] Dispatcher started with %d workers (threat threshold: %d)\n", poolSize, threatThreshold)
	fmt.Println()

	// --- Read the file line-by-line and feed the pool ---
	scanner := bufio.NewScanner(input)

	// Increase the default 64KB buffer for safety with long lines
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	processed := 0
	skipped := 0

	start := time.Now()

	for scanner.Scan() {
		line := scanner.Bytes()

		// Validate through ParseLine before submitting
		_, err := parser.ParseLine(line)
		if err != nil {
			skipped++
			continue
		}

		// Copy the line — scanner reuses its internal buffer
		payload := make([]byte, len(line))
		copy(payload, line)

		dispatcher.Submit(worker.Job{Payload: payload})
		processed++
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Scanner error: %v\n", err)
	}

	// --- Shutdown and measure ---
	dispatcher.Stop()
	elapsed := time.Since(start)

	rate := float64(processed) / elapsed.Seconds()
	trackedIPs, threats := detector.Summary()

	fmt.Println()
	fmt.Println("════════════════════════════════════════")
	fmt.Println("          LogFlow Engine Report         ")
	fmt.Println("════════════════════════════════════════")
	fmt.Printf("  Processed : %d entries\n", processed)
	fmt.Printf("  Skipped   : %d entries\n", skipped)
	fmt.Printf("  Duration  : %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("  Throughput: %.0f logs/sec\n", rate)
	fmt.Println("────────────────────────────────────────")
	fmt.Printf("  IPs tracked : %d\n", trackedIPs)
	fmt.Printf("  Threats     : %d brute-force alerts\n", threats)
	fmt.Println("════════════════════════════════════════")

	if rate >= 10_000 {
		fmt.Println("  ✓ PASS — meets 10,000 logs/sec target")
	} else {
		fmt.Println("  ✗ FAIL — below 10,000 logs/sec target")
	}
}
