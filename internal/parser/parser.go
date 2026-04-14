package parser

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/varkuru/logflow/internal/worker"
)

// StreamParser reads JSON log objects one at a time from a reader,
// minimizing memory overhead by avoiding loading the entire file into memory.
type StreamParser struct {
	decoder *json.Decoder
}

// NewStreamParser creates a parser that wraps the given reader with a json.Decoder.
func NewStreamParser(r io.Reader) *StreamParser {
	dec := json.NewDecoder(r)
	return &StreamParser{decoder: dec}
}

// Parse reads log entries from the stream and submits each one as a Job
// to the provided Dispatcher. It stops when the stream is exhausted.
func (p *StreamParser) Parse(dispatcher *worker.Dispatcher) (int, error) {
	count := 0

	for p.decoder.More() {
		// Decode into raw JSON first so we can use ParseLine for validation
		var raw json.RawMessage
		if err := p.decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return count, fmt.Errorf("decode error at entry %d: %w", count+1, err)
		}

		// Validate the entry through ParseLine before submitting
		if _, err := ParseLine(raw); err != nil {
			// Skip malformed entries — log the error and continue
			fmt.Printf("[WARN] skipping entry %d: %v\n", count+1, err)
			continue
		}

		dispatcher.Submit(worker.Job{Payload: raw})
		count++
	}

	return count, nil
}
