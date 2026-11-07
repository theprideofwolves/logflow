package analysis

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/theprideofwolves/logflow/internal/parser"
	"io"
	"net/netip"
	"strings"
	"sync"
	"time"
)

type Options struct {
	Threshold, Workers, MaxIPs, MaxAlerts int
	Window                                time.Duration
}

func DefaultOptions() Options { return Options{6, 4, 10000, 1000, 10 * time.Minute} }
func (o Options) Validate() error {
	if o.Threshold < 2 || o.Threshold > 100 || o.Workers < 1 || o.Workers > 32 || o.MaxIPs < 1 || o.MaxIPs > 100000 || o.MaxAlerts < 1 || o.MaxAlerts > 1000 || o.Window < time.Second || o.Window > 24*time.Hour {
		return fmt.Errorf("threshold 2-100, workers 1-32, IPs 1-100000, alerts 1-1000, window 1s-24h required")
	}
	return nil
}

type Evidence struct {
	Line              int
	Timestamp         time.Time
	Message, SourceIP string
}
type Alert struct {
	ID, IP     string
	Start, End time.Time
	Count      int
	Evidence   []Evidence
}
type Rejection struct {
	Line   int
	Reason string
}
type Report struct {
	Version, InputHash                      string
	At                                      time.Time
	Options                                 Options
	Complete                                bool
	Accepted, Rejected, Unattributed, Lines int
	Rejections                              []Rejection
	Alerts                                  []Alert
	Limit                                   string
}
type candidate struct {
	entry parser.LogEntry
	err   error
	line  int
}
type stream struct {
	events  []Evidence
	alerted bool
}

func Process(ctx context.Context, input io.Reader, o Options) (*Report, error) {
	if e := o.Validate(); e != nil {
		return nil, e
	}
	r := &Report{Version: "logflow/v2", Options: o, At: time.Now().UTC(), Complete: true, Alerts: []Alert{}, Rejections: []Rejection{}}
	hash := sha256.New()
	scanner := bufio.NewScanner(io.TeeReader(input, hash))
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	states := map[string]*stream{}
	var last time.Time
	var sweep time.Time
	reject := func(line int, reason string) {
		r.Rejected++
		r.Complete = false
		if len(r.Rejections) < 1000 {
			r.Rejections = append(r.Rejections, Rejection{line, reason})
		}
	}
	stop := false
	for !stop {
		if e := ctx.Err(); e != nil {
			r.Complete = false
			r.Limit = "cancelled; partial evidence"
			break
		}
		// Bound raw memory and preserve input order after concurrent parsing.
		batch := [][]byte{}
		batchBytes := 0
		for len(batch) < 256 && batchBytes < 4<<20 && scanner.Scan() {
			batch = append(batch, append([]byte(nil), scanner.Bytes()...))
			batchBytes += len(scanner.Bytes())
		}
		if len(batch) == 0 {
			break
		}
		parsed := make([]candidate, len(batch))
		jobs := make(chan int)
		var wg sync.WaitGroup
		for worker := 0; worker < o.Workers; worker++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range jobs {
					e, err := parser.ParseLine(batch[i])
					parsed[i] = candidate{e, err, r.Lines + i + 1}
				}
			}()
		}
		for i := range batch {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		for _, c := range parsed {
			r.Lines++
			if c.err != nil {
				reject(c.line, "invalid JSON or required timestamp/level/message")
				continue
			}
			e := c.entry
			if !last.IsZero() && e.Timestamp.Before(last) {
				reject(c.line, "out-of-order event; sort by timestamp and rerun")
				continue
			}
			if e.SourceIP != "" {
				ip, err := netip.ParseAddr(e.SourceIP)
				if err != nil {
					reject(c.line, "invalid source IP")
					continue
				}
				e.SourceIP = ip.Unmap().String()
			}
			last = e.Timestamp
			r.Accepted++
			if e.SourceIP == "" {
				r.Unattributed++
				continue
			}
			if e.Message != "Failed login attempt" {
				continue
			}
			cutoff := e.Timestamp.Add(-o.Window)
			if sweep.IsZero() || e.Timestamp.Sub(sweep) >= time.Second {
				for ip, s := range states {
					for len(s.events) > 0 && s.events[0].Timestamp.Before(cutoff) {
						s.events = s.events[1:]
					}
					if len(s.events) == 0 {
						delete(states, ip)
					}
				}
				sweep = e.Timestamp
			}
			s := states[e.SourceIP]
			if s == nil {
				if len(states) >= o.MaxIPs {
					r.Complete = false
					r.Limit = "active IP limit reached; narrow scope or increase configured limit"
					stop = true
					break
				}
				s = &stream{}
				states[e.SourceIP] = s
			}
			for len(s.events) > 0 && s.events[0].Timestamp.Before(cutoff) {
				s.events = s.events[1:]
			}
			if len(s.events) < o.Threshold {
				s.alerted = false
			}
			s.events = append(s.events, Evidence{c.line, e.Timestamp, e.Message, e.SourceIP})
			if len(s.events) > o.Threshold {
				s.events = s.events[len(s.events)-o.Threshold:]
			}
			if len(s.events) >= o.Threshold && !s.alerted {
				if len(r.Alerts) >= o.MaxAlerts {
					r.Complete = false
					r.Limit = "alert limit reached; partial evidence"
					stop = true
					break
				}
				start := s.events[0].Timestamp
				id := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%d|%s", e.SourceIP, start.Format(time.RFC3339Nano), e.Timestamp.Format(time.RFC3339Nano), o.Threshold, o.Window)))
				r.Alerts = append(r.Alerts, Alert{hex.EncodeToString(id[:]), e.SourceIP, start, e.Timestamp, len(s.events), append([]Evidence(nil), s.events...)})
				s.alerted = true
			}
		}
	}
	if e := scanner.Err(); e != nil {
		r.Complete = false
		r.Limit = "input read failed or line exceeds 1 MiB; counts cover processed lines only"
	}
	r.InputHash = hex.EncodeToString(hash.Sum(nil))
	if stop || r.Limit != "" {
		r.InputHash = "partial-prefix:" + r.InputHash
	}
	// Final alert identity includes input evidence identity and rule configuration.
	for i := range r.Alerts {
		v := sha256.Sum256([]byte(r.InputHash + "|" + r.Alerts[i].ID))
		r.Alerts[i].ID = hex.EncodeToString(v[:])
	}
	return r, nil
}
func Demo() string {
	var b strings.Builder
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&b, "{\"timestamp\":\"2026-11-06T12:00:%02dZ\",\"level\":\"WARN\",\"message\":\"Failed login attempt\",\"source_ip\":\"192.0.2.10\"}\n", i)
	}
	return b.String()
}
