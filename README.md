# LogFlow 🚀

A concurrent log processing engine built in Go. LogFlow ingests structured JSON logs at high speed, distributes them across a fixed worker pool, and detects brute-force attacks in real time using mutex-protected pattern analysis.

---

## Performance

LogFlow was benchmarked at **447,000+ logs/sec** through a combination of three design choices:

| Technique | Impact |
|---|---|
| **Fixed Worker Pool** | 100 goroutines process jobs in parallel without spawning new threads |
| **Streaming Parser** | `bufio.Scanner` reads one line at a time — memory stays flat regardless of file size |
| **Buffered Channel** | A 1,000-job queue absorbs bursts and prevents backpressure stalls |

```
════════════════════════════════════════
          LogFlow Engine Report
════════════════════════════════════════
  Processed : 10,060 entries
  Skipped   : 0 entries
  Duration  : 22ms
  Throughput: 447,405 logs/sec
────────────────────────────────────────
  IPs tracked : 5,799
  Threats     : 3 brute-force alerts
════════════════════════════════════════
  ✓ PASS — meets 10,000 logs/sec target
```

---

## Architecture

Logs flow through a five-stage pipeline: ingestion, validation, dispatch, concurrent analysis, and threat reporting.

![Architecture](./assets/architecture.png)

Each worker goroutine receives a job from the dispatcher channel, unmarshals the payload, and passes the entry through a thread-safe threat detector. If an IP exceeds the failure threshold, an alert fires immediately.

![Logic Flow](./assets/logic_flow.png)

---

## Usage

### Process a log file

```bash
./logflow access_logs.json
```

### Pipe from another source

```bash
cat /var/log/auth.json | ./logflow
```

```bash
kubectl logs deployment/api-gateway | ./logflow
```

LogFlow reads from `stdin` when no file argument is provided. The engine report prints after the stream closes.

### Installation

```bash
git clone https://github.com/theprideofwolves/logflow
cd logflow
go build -o logflow ./cmd/logflow
```

### Expected Log Format

One JSON object per line:

```json
{
  "timestamp": "2026-04-14T21:27:00.507Z",
  "level": "WARN",
  "source_ip": "10.0.0.66",
  "user_id": "user_0001",
  "message": "Failed login attempt"
}
```

| Field | Type | Required |
|---|---|---|
| `timestamp` | RFC 3339 | Yes |
| `level` | `INFO` · `WARN` · `ERROR` | Yes |
| `message` | string | Yes |
| `source_ip` | string | No |
| `user_id` | string | No |

---

## Design Goals

**Memory efficiency.** The streaming parser reads one line at a time and discards it after processing. There is no point where the entire file is held in memory.

**CPU discipline.** The worker pool is fixed at initialization. No new goroutines are spawned during processing, preventing thread exhaustion under load.

**Thread-safe detection.** The `ThreatDetector` uses a `sync.Mutex` to guard a shared `map[string]int` that all 100 workers write to concurrently. Each IP is tracked independently, and alerts fire exactly once per IP when the failure count exceeds the threshold.

**Clean separation.** Workers accept a `ProcessFunc` callback at startup. The detection logic is injected from `main.go` without any circular dependencies between packages.

---

## Project Structure

```
logflow/
├── cmd/logflow/
│   └── main.go               # CLI entry point
├── internal/
│   ├── parser/
│   │   ├── json_parser.go     # LogEntry struct and ParseLine validator
│   │   └── parser.go          # Streaming JSON decoder
│   ├── worker/
│   │   └── pool.go            # Job, Worker, Dispatcher, ProcessFunc
│   └── patterns/
│       └── detector.go        # Mutex-protected brute-force detector
└── assets/
    ├── architecture.png
    └── logic_flow.png
```

---

## License

MIT

<p align="center">Built with Go, goroutines, and zero tolerance for brute force.</p>
