package analysis

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

func TestExpiredIPDoesNotConsumeCapacityBetweenSweeps(t *testing.T) {
	o := DefaultOptions()
	o.MaxIPs = 2
	o.Window = time.Second
	var input strings.Builder
	// At 1.5s the IP seen at 0.4s is expired, but the periodic sweep ran at 1.1s.
	for i, ms := range []int{0, 400, 1100, 1500} {
		fmt.Fprintf(&input, "{\"timestamp\":%q,\"level\":\"WARN\",\"message\":\"Failed login attempt\",\"source_ip\":\"192.0.2.%d\"}\n", time.UnixMilli(int64(ms)).UTC().Format(time.RFC3339Nano), i+1)
	}
	r, err := Process(context.Background(), strings.NewReader(input.String()), o)
	if err != nil || !r.Complete || r.Accepted != 4 || r.Limit != "" {
		t.Fatalf("expired IP caused false limit: %+v, %v", r, err)
	}
}

func TestIPNormalizationAndExactMessage(t *testing.T) {
	o := DefaultOptions()
	o.Threshold = 2
	for _, tc := range []struct {
		name, first, second, message string
		alerts                       int
	}{
		{"mapped IPv4", "::ffff:192.0.2.1", "192.0.2.1", "Failed login attempt", 1},
		{"IPv6 spelling", "2001:0db8:0:0:0:0:0:1", "2001:db8::1", "Failed login attempt", 1},
		{"separate sources", "192.0.2.1", "192.0.2.2", "Failed login attempt", 0},
		{"case matters", "192.0.2.1", "192.0.2.1", "failed login attempt", 0},
		{"whitespace matters", "192.0.2.1", "192.0.2.1", "Failed login attempt ", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := Process(context.Background(), strings.NewReader(event(0, tc.first, tc.message)+event(0, tc.second, tc.message)), o)
			if err != nil || !r.Complete || len(r.Alerts) != tc.alerts || r.Accepted != 2 {
				t.Fatal(r, err)
			}
			if tc.alerts > 0 && (r.Alerts[0].Evidence[0].SourceIP != r.Alerts[0].IP || r.Alerts[0].Evidence[1].SourceIP != r.Alerts[0].IP) {
				t.Fatal("noncanonical evidence", r.Alerts)
			}
		})
	}
}

type failingReader struct{ io.Reader }

func (r failingReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return n, err
}

type cancellingReader struct {
	io.Reader
	cancel context.CancelFunc
}

func (r cancellingReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.cancel()
	return n, err
}

func TestReadFailureAndCancellationDuringInput(t *testing.T) {
	r, err := Process(context.Background(), failingReader{strings.NewReader(Demo())}, DefaultOptions())
	if err != nil || r.Complete || r.Accepted != 8 || len(r.Alerts) != 1 || !strings.HasPrefix(r.InputHash, "partial-prefix:") || !strings.Contains(r.Limit, "input read failed") {
		t.Fatal(r, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err = Process(ctx, cancellingReader{strings.NewReader(Demo()), cancel}, DefaultOptions())
	if err != nil || r.Complete || !strings.Contains(r.Limit, "cancelled") || !strings.HasPrefix(r.InputHash, "partial-prefix:") || r.Accepted+r.Rejected != r.Lines {
		t.Fatal(r, err)
	}
}

func TestOrderedRejectionsAndLineAccounting(t *testing.T) {
	input := event(10, "192.0.2.1", "Failed login attempt") + "{}\n" + event(11, "invalid", "Failed login attempt") + event(9, "192.0.2.1", "Failed login attempt") + event(12, "", "Failed login attempt")
	r, err := Process(context.Background(), strings.NewReader(input), DefaultOptions())
	if err != nil || r.Complete || r.Accepted != 2 || r.Rejected != 3 || r.Lines != 5 || r.Unattributed != 1 {
		t.Fatal(r, err)
	}
	for i, line := range []int{2, 3, 4} {
		if r.Rejections[i].Line != line {
			t.Fatal(r.Rejections)
		}
	}
}

func TestOptionsBoundaries(t *testing.T) {
	for _, change := range []func(*Options){
		func(o *Options) { o.Threshold = 1 }, func(o *Options) { o.Threshold = 101 },
		func(o *Options) { o.Workers = 0 }, func(o *Options) { o.Workers = 33 },
		func(o *Options) { o.MaxIPs = 0 }, func(o *Options) { o.MaxIPs = 100001 },
		func(o *Options) { o.MaxAlerts = 0 }, func(o *Options) { o.MaxAlerts = 1001 },
		func(o *Options) { o.Window = time.Second - time.Nanosecond }, func(o *Options) { o.Window = 24*time.Hour + time.Nanosecond },
	} {
		o := DefaultOptions()
		change(&o)
		if _, err := Process(context.Background(), strings.NewReader(""), o); err == nil {
			t.Fatal("invalid options accepted", o)
		}
	}
	for _, o := range []Options{{2, 1, 1, 1, time.Second}, {100, 32, 100000, 1000, 24 * time.Hour}} {
		if err := o.Validate(); err != nil {
			t.Fatal(o, err)
		}
	}
}

func FuzzProcessReportInvariants(f *testing.F) {
	for _, seed := range []string{"", Demo(), "bad\n{}\n", event(1, "bad", "Failed login attempt")} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 64<<10 {
			t.Skip()
		}
		o := DefaultOptions()
		o.Workers = 1
		o.MaxIPs = 8
		o.MaxAlerts = 4
		r, err := Process(context.Background(), strings.NewReader(input), o)
		if err != nil {
			t.Fatal(err)
		}
		if r.Accepted+r.Rejected != r.Lines || len(r.Rejections) > r.Rejected || len(r.Alerts) > o.MaxAlerts || (r.Complete && (r.Rejected > 0 || r.Limit != "")) {
			t.Fatalf("inconsistent report: %+v", r)
		}
		for _, a := range r.Alerts {
			if len(a.Evidence) != o.Threshold || a.Count != o.Threshold || a.End.Before(a.Start) || a.End.Sub(a.Start) > o.Window {
				t.Fatal(a)
			}
		}
	})
}
