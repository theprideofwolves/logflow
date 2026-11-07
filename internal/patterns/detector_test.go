package patterns

import (
	"github.com/varkuru/logflow/internal/parser"
	"sync"
	"testing"
)

func TestConcurrentFailuresAlertOnce(t *testing.T) {
	d := NewThreatDetector(5)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.Analyze(parser.LogEntry{SourceIP: "192.0.2.10", Message: "Failed login attempt"})
		}()
	}
	wg.Wait()
	ips, alerts := d.Summary()
	if ips != 1 || alerts != 1 {
		t.Fatalf("ips=%d alerts=%d", ips, alerts)
	}
}
func TestThresholdAndMissingIP(t *testing.T) {
	d := NewThreatDetector(5)
	for i := 0; i < 20; i++ {
		d.Analyze(parser.LogEntry{Level: "ERROR"})
	}
	for i := 0; i < 5; i++ {
		d.Analyze(parser.LogEntry{SourceIP: "192.0.2.10", Level: "ERROR"})
	}
	_, alerts := d.Summary()
	if alerts != 0 {
		t.Fatal("early or missing-IP alert")
	}
	d.Analyze(parser.LogEntry{SourceIP: "192.0.2.10", Level: "ERROR"})
	_, alerts = d.Summary()
	if alerts != 1 {
		t.Fatal("sixth failure did not alert")
	}
}
