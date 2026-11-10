package analysis

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// BenchmarkProcess measures in-memory parsing/reduction including sensitivity,
// excluding fixture generation, disk I/O, UI, JSON serialization and Jira.
func BenchmarkProcess(b *testing.B) {
	for _, scenario := range []struct{ lines, ips int }{{10000, 1}, {100000, 1000}} {
		var fixture strings.Builder
		for i := 0; i < scenario.lines; i++ {
			ip := i % scenario.ips
			fmt.Fprintf(&fixture, "{\"timestamp\":%q,\"level\":\"WARN\",\"message\":\"Failed login attempt\",\"source_ip\":\"192.0.%d.%d\"}\n", time.Unix(0, int64(i)*int64(time.Millisecond)).UTC().Format(time.RFC3339Nano), ip/256, ip%256)
		}
		input := fixture.String()
		for _, workers := range []int{1, 4} {
			b.Run(fmt.Sprintf("lines=%d/ips=%d/workers=%d", scenario.lines, scenario.ips, workers), func(b *testing.B) {
				o := DefaultOptions()
				o.Workers = workers
				b.SetBytes(int64(len(input)))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					r, err := Process(context.Background(), strings.NewReader(input), o)
					if err != nil || !r.Complete || r.Accepted != scenario.lines || len(r.Alerts) != scenario.ips {
						b.Fatalf("incorrect benchmark result: %v %+v", err, r)
					}
				}
				b.ReportMetric(float64(scenario.lines)*float64(b.N)/b.Elapsed().Seconds(), "records/s")
			})
		}
	}
}
