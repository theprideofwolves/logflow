# Interview walkthrough: Concurrent security log processing

## Explain it in 45 seconds
I built a streaming Go log processor with a fixed worker pool and buffered queue, plus mutex-protected per-IP counters. It flags repeated failure events and reports processing totals. I added regression tests and excluded missing-IP events from per-IP detection.

## Demonstrate it
1. Run `go run ./cmd/logflow testdata/demo.jsonl`.
2. Expect 8 processed entries, 1 skipped entry, and 1 alert for 192.0.2.10.
3. Six failures for that IP trigger one alert. A normal event and a missing-IP error are processed without another per-IP alert.
4. Run `go test ./...` to check concurrent alert deduplication, validation, and queue draining.

## Questions to be ready for
- Why 100 workers? A fixed design setting, not a benchmark-proven optimum. It is a tuning parameter for future workload measurements.
- Why a buffered queue? Absorb a limited burst; when full, the sender blocks and applies backpressure.
- Why a mutex? Shared maps and alert counters are accessed by multiple goroutines.
- Threshold semantics? More than five matching failures, once per IP per run. There is no time window or expiry.
- Limits? Detection uses exact messages or ERROR level; memory grows with distinct failing IPs. Worker processing order is not input order. This is a heuristic, not proof of brute force.
- Performance? Reported throughput is observed per run. A tiny fixture cannot validate a throughput target.

## Timeline and resume consistency
The submitted resume describes the original project's core behavior. These subsequent improvements strengthen the same project. If asked, distinguish the original implementation from later hardening. Practice the demo and inspect the source before describing decisions as your own. Do not claim production deployment, measured business impact, or autonomous agents from this repository.
