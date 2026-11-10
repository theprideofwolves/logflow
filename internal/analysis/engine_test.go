package analysis

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func event(sec int, ip, message string) string {
	return fmt.Sprintf("{\"timestamp\":%q,\"level\":\"ERROR\",\"message\":%q,\"source_ip\":%q}\n", time.Unix(int64(sec), 0).UTC().Format(time.RFC3339), message, ip)
}
func TestWindowBoundariesAndRearm(t *testing.T) {
	o := DefaultOptions()
	o.Threshold = 2
	o.Window = 10 * time.Second
	for _, tc := range []struct {
		name   string
		times  []int
		alerts int
	}{{"inclusive", []int{0, 10}, 1}, {"outside", []int{0, 11}, 0}, {"dedup episode", []int{0, 1, 2, 3}, 1}, {"rearm", []int{0, 1, 20, 21}, 2}} {
		t.Run(tc.name, func(t *testing.T) {
			var b strings.Builder
			for _, s := range tc.times {
				b.WriteString(event(s, "192.0.2.1", "Failed login attempt"))
			}
			r, e := Process(context.Background(), strings.NewReader(b.String()), o)
			if e != nil || !r.Complete || len(r.Alerts) != tc.alerts {
				t.Fatal(r, e)
			}
			for _, a := range r.Alerts {
				if len(a.Evidence) != o.Threshold || a.Count != o.Threshold {
					t.Fatal(a)
				}
			}
		})
	}
}
func TestInvalidAndUnattributed(t *testing.T) {
	input := event(10, "192.0.2.1", "Failed login attempt") + event(9, "192.0.2.1", "Failed login attempt") + "bad\n" + event(11, "bad-ip", "Failed login attempt") + event(12, "", "Failed login attempt") + event(13, "192.0.2.1", "database unavailable")
	r, e := Process(context.Background(), strings.NewReader(input), DefaultOptions())
	if e != nil || r.Complete || r.Rejected != 3 || r.Accepted != 3 || r.Unattributed != 1 || r.Lines != 6 || len(r.Alerts) != 0 {
		t.Fatal(r, e)
	}
}
func TestLimitsAndCancellation(t *testing.T) {
	o := DefaultOptions()
	o.MaxIPs = 1
	r, e := Process(context.Background(), strings.NewReader(event(1, "192.0.2.1", "Failed login attempt")+event(2, "192.0.2.2", "Failed login attempt")), o)
	if e != nil || r.Complete || !strings.Contains(r.Limit, "IP limit") {
		t.Fatal(r, e)
	}
	o.MaxAlerts = 1
	o.Threshold = 2
	o.Window = time.Second
	r, _ = Process(context.Background(), strings.NewReader(event(1, "192.0.2.1", "Failed login attempt")+event(2, "192.0.2.1", "Failed login attempt")+event(10, "192.0.2.1", "Failed login attempt")+event(11, "192.0.2.1", "Failed login attempt")), o)
	if r.Complete || len(r.Alerts) != 1 {
		t.Fatal(r)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, _ = Process(ctx, strings.NewReader(Demo()), o)
	if r.Complete || r.Limit == "" {
		t.Fatal(r)
	}
	r, _ = Process(context.Background(), strings.NewReader(strings.Repeat("x", 2<<20)), o)
	if r.Complete || r.Limit == "" {
		t.Fatal("truncation looked complete")
	}
}
func TestWorkersDeterministic(t *testing.T) {
	o := DefaultOptions()
	var id string
	for _, n := range []int{1, 4, 32} {
		o.Workers = n
		r, e := Process(context.Background(), strings.NewReader(Demo()), o)
		if e != nil || len(r.Alerts) != 1 {
			t.Fatal(r, e)
		}
		if id != "" && r.Alerts[0].ID != id {
			t.Fatal("worker order changed decision")
		}
		id = r.Alerts[0].ID
	}
	o.Workers = 0
	if _, e := Process(context.Background(), strings.NewReader(""), o); e == nil {
		t.Fatal("invalid options")
	}
}
func TestRejectDetailCap(t *testing.T) {
	r, e := Process(context.Background(), strings.NewReader(strings.Repeat("bad\n", 1200)), DefaultOptions())
	if e != nil || r.Rejected != 1200 || len(r.Rejections) != 1000 || r.Accepted+r.Rejected != r.Lines {
		t.Fatal(r, e)
	}
}

func TestDefaultRuleSyntheticScenarios(t *testing.T) {
	for _, tc := range []struct {
		name    string
		times   []int
		message string
		alerts  int
	}{
		{"inclusive ten minutes", []int{0, 120, 240, 360, 480, 600}, "Failed login attempt", 1},
		{"below six", []int{0, 120, 240, 360, 480}, "Failed login attempt", 0},
		{"expired first event", []int{0, 120, 240, 360, 480, 601}, "Failed login attempt", 0},
		{"generic errors", []int{0, 120, 240, 360, 480, 600}, "database unavailable", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var input strings.Builder
			for _, sec := range tc.times {
				input.WriteString(event(sec, "192.0.2.10", tc.message))
			}
			r, err := Process(context.Background(), strings.NewReader(input.String()), DefaultOptions())
			if err != nil || !r.Complete || r.Accepted != len(tc.times) || len(r.Alerts) != tc.alerts {
				t.Fatalf("report=%+v err=%v", r, err)
			}
			if tc.alerts == 1 {
				a := r.Alerts[0]
				if a.IP != "192.0.2.10" || a.Count != 6 || len(a.Evidence) != 6 || a.End.Sub(a.Start) != 10*time.Minute {
					t.Fatal(a)
				}
				for i, evidence := range a.Evidence {
					if evidence.Line != i+1 || evidence.SourceIP != a.IP || evidence.Message != tc.message || evidence.Timestamp.Unix() != int64(tc.times[i]) {
						t.Fatal(evidence)
					}
				}
			}
		})
	}
}
