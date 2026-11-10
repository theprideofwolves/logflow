package analysis

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ReplayResult struct {
	Name, Explanation              string
	ExpectedAlerts, ObservedAlerts int
	Complete, MatchesExpectation   bool
}

// Replay evaluates fixed synthetic scenarios, not live attacks or field accuracy.
func Replay(ctx context.Context) ([]ReplayResult, error) {
	scenarios := []struct {
		name, why   string
		spacing     time.Duration
		distributed bool
		expected    int
	}{
		{"concentrated failures", "Six failures from one IP within ten minutes support the rule.", time.Minute, false, 1},
		{"slow attempts", "Six failures spread twelve minutes apart evade this ten-minute window.", 12 * time.Minute, false, 0},
		{"distributed sources", "One failure from each of six IPs evades per-IP aggregation.", time.Minute, true, 0},
		{"benign repeated failures", "The same failure pattern can arise from a stale client credential; this rule cannot infer intent.", time.Minute, false, 1},
	}
	results := make([]ReplayResult, 0, len(scenarios))
	for _, s := range scenarios {
		var input strings.Builder
		for i := 0; i < 6; i++ {
			ip := "192.0.2.10"
			if s.distributed {
				ip = fmt.Sprintf("192.0.2.%d", i+1)
			}
			timestamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * s.spacing)
			fmt.Fprintf(&input, "{\"timestamp\":%q,\"level\":\"WARN\",\"message\":\"Failed login attempt\",\"source_ip\":%q}\n", timestamp.Format(time.RFC3339Nano), ip)
		}
		r, err := Process(ctx, strings.NewReader(input.String()), DefaultOptions())
		if err != nil {
			return nil, err
		}
		observed := len(r.Alerts)
		results = append(results, ReplayResult{s.name, s.why, s.expected, observed, r.Complete, r.Complete && observed == s.expected})
	}
	return results, nil
}
