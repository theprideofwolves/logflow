# LogFlow
### Concurrent JSON log analysis in Go
A streaming command-line tool that reads newline-delimited JSON logs from a file or stdin and flags repeated failure events by source IP.

## What I built
- A fixed pool of 100 workers connected through a buffered queue of 1,000 jobs.
- Line-by-line input validation for timestamp, level, and message fields.
- Mutex-protected failure counters and one alert per IP after more than five matching failures.
- A final report of processed and skipped entries, elapsed time, measured throughput, tracked IPs, and alert totals.

## Architecture
`file / stdin -> validation -> buffered queue -> workers -> shared detector -> summary`

![Architecture](assets/architecture.png)

## Build and run
Requires Go 1.25.5 or newer, matching `go.mod`.

```bash
git clone https://github.com/theprideofwolves/logflow.git
cd logflow
go build -o logflow ./cmd/logflow
./logflow access_logs.json
```

On Windows, build with `-o logflow.exe` and run `.\logflow.exe access_logs.json`. With no file argument, LogFlow reads stdin and prints its summary when input closes.

### Input format
One JSON object per line:
```json
{"timestamp":"2026-11-05T12:00:00Z","level":"WARN","source_ip":"192.0.2.10","user_id":"demo-user","message":"Failed login attempt"}
```

`timestamp`, `level`, and `message` are required. `source_ip` and `user_id` are optional in the parser; include `source_ip` for meaningful per-IP detection.

### Detection behavior
The detector counts exact messages `Failed login attempt`, `Unauthorized API access detected`, `Rate limit exceeded`, and `Account locked after repeated failures`, plus entries whose level is exactly `ERROR`.

The sixth matching failure for an IP triggers an alert. Counts accumulate over one run with no time window or expiry. Missing IPs share an empty-string counter. Alerts indicate possible suspicious activity and require investigation.

## Engineering notes
The queue bounds pending work and applies backpressure when full. The scanner limits each line to 1 MiB. Memory also grows with the number of distinct IPs retained by the detector.

The CLI reports observed throughput for each run. No portable throughput guarantee is claimed; results depend on input, machine, and execution conditions.

## Source map
| Path | Responsibility |
| --- | --- |
| `cmd/logflow/main.go` | Input, validation, dispatch, final report |
| `internal/parser/` | JSON models and parsing |
| `internal/worker/` | Worker lifecycle and buffered queue |
| `internal/patterns/` | Shared counters and alert logic |

## Resume bullets
- Built a streaming Go log processor that validates JSON events from files or stdin and distributes analysis across 100 workers through a 1,000-job buffered queue.
- Implemented mutex-protected per-IP failure counters, threshold alerts for possible brute-force activity, and processing summaries covering skipped entries, throughput, and alert totals.
