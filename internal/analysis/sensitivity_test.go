package analysis

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSensitivityScopeAndInclusiveBoundaries(t *testing.T) {
	var b strings.Builder
	for _, sec := range []int{0, 120, 240, 360, 480, 600} {
		b.WriteString(event(sec, "192.0.2.1", "Failed login attempt"))
	}
	r, err := Process(context.Background(), strings.NewReader(b.String()), DefaultOptions())
	if err != nil || len(r.Alerts) != 1 {
		t.Fatal(r, err)
	}
	a := r.Alerts[0]
	if len(a.Sensitivity) != 9 {
		t.Fatal(a.Sensitivity)
	}
	for _, cell := range a.Sensitivity {
		wantPeak := 6
		if cell.Window == 5*time.Minute {
			wantPeak = 3
		}
		if cell.Peak != wantPeak || cell.Supported != (wantPeak >= cell.Threshold) {
			t.Fatal(cell)
		}
	}
	// Additional input can support a higher threshold although the six retained
	// trigger records cannot: this is why the scope label is essential.
	b.WriteString(event(600, "192.0.2.1", "Failed login attempt"))
	o := DefaultOptions()
	o.Threshold = 7
	full, err := Process(context.Background(), strings.NewReader(b.String()), o)
	if err != nil || len(full.Alerts) != 1 {
		t.Fatal(full, err)
	}
	for _, cell := range a.Sensitivity {
		if cell.Threshold == 7 && cell.Supported {
			t.Fatal("invented unretained evidence")
		}
	}
}

func TestSensitivityBounds(t *testing.T) {
	for _, o := range []Options{{2, 1, 1, 1, time.Second}, {100, 1, 1, 1, 24 * time.Hour}} {
		cells := Sensitivity(nil, o)
		if len(cells) != 4 {
			t.Fatal(cells)
		}
		for _, c := range cells {
			if c.Supported || c.Peak != 0 || c.Threshold < 2 || c.Threshold > 100 || c.Window < time.Second || c.Window > 24*time.Hour {
				t.Fatal(c)
			}
		}
	}
}

func TestSensitivityMatchesIndependentIntervalCounts(t *testing.T) {
	for _, times := range [][]int{{0, 0, 1, 2, 600, 601}, {0, 300, 600, 900, 1200}, {0, 1, 2, 3, 4, 5}} {
		var evidence []Evidence
		for i, s := range times {
			evidence = append(evidence, Evidence{Line: i + 1, Timestamp: time.Unix(int64(s), 0)})
		}
		for _, cell := range Sensitivity(evidence, DefaultOptions()) {
			peak := 0
			for _, end := range evidence {
				count := 0
				for _, e := range evidence {
					if !e.Timestamp.Before(end.Timestamp.Add(-cell.Window)) && !e.Timestamp.After(end.Timestamp) {
						count++
					}
				}
				if count > peak {
					peak = count
				}
			}
			if cell.Peak != peak || cell.Supported != (peak >= cell.Threshold) {
				t.Fatal(times, cell, peak)
			}
		}
	}
}

func TestReplayDocumentsBlindSpots(t *testing.T) {
	results, err := Replay(context.Background())
	if err != nil || len(results) != 4 {
		t.Fatal(results, err)
	}
	for i, r := range results {
		expected := []int{1, 0, 0, 1}[i]
		if !r.Complete || !r.MatchesExpectation || r.ObservedAlerts != expected || r.ExpectedAlerts != expected {
			t.Fatal(r)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results, err = Replay(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Complete || r.MatchesExpectation {
			t.Fatal("cancelled replay looked complete", r)
		}
	}
}
