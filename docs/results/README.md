# Verification results — 2026-11-10

All 10 recorded automated checks passed. 47 top-level tests and 55 subtests passed; 0 tests skipped.
Statement coverage: **77.7%**, measured with race detection. Coverage is not a correctness or attack-detection score.

Environment: go version go1.25.5 windows/amd64; Windows 11, AMD64;
16 logical CPUs available, Go constrained to GOMAXPROCS=4. CPU: AMD64 Family 25 Model 80 Stepping 0, AuthenticAMD.

Reproduce: `python tools/verify.py --date YYYY-MM-DD`. Requires Go, a race-compatible
C compiler and Python 3.10+. All application data is temporary and browser opening
is disabled. Jira tests use local TLS mocks; no credentials, paid APIs or cloud resources.

## In-memory processing benchmarks

| Input / sources / workers | Median records/s | Observed range | Allocated bytes/analysis |
| --- | ---: | ---: | ---: |
| lines=10000/ips=1/workers=1 | 255,341 | 251,093–257,658 | 8,132,130 |
| lines=10000/ips=1/workers=4 | 671,294 | 659,574–674,518 | 8,143,258 |
| lines=100000/ips=1000/workers=1 | 252,922 | 251,362–253,062 | 85,044,264 |
| lines=100000/ips=1000/workers=4 | 635,075 | 622,688–644,341 | 85,121,930 |

Each row summarizes three repeated samples, three complete analyses per sample.
Fixture generation is excluded. Times include parsing, ordered reduction, hashing,
evidence and sensitivity; exclude disk I/O, serialization, UI and Jira.
**Allocated bytes are cumulative Go allocations, not peak process memory.**
All records are matching failures spaced 1 ms apart, round-robin over the stated
number of IPs. Defaults apply except worker count. Each analysis must finish
complete with the exact record count and one alert per source. Benchmarks use
normal builds, not race-instrumented builds. Do not extrapolate to production workloads.

## Detection behavior

Four fixed synthetic replays agree with expectations: concentrated failures **1**
alert; slow attempts **0**; distributed sources **0**; benign repeated failures **1**.
This demonstrates two blind spots and a benign alert, **not 100% detection accuracy**.
The benign label is supplied by the fixture, not inferred from log contents.

## Functional evidence and limits

See [the functionality matrix](../FUNCTIONALITY.md), [every passed test](results.json),
[raw benchmark samples](benchmarks.txt), and [the manual checklist](../../TESTING.md).
Manual observations are not claimed. Linux was cross-compiled, not executed locally;
GitHub's separate OS jobs provide additional evidence only after they actually pass.

Source fingerprint (LF-normalized code/test/tool/workflow files): `b7905909063f2115d408e177e523efd8e3762c2c8a026af8c96240c14b359890`.
The per-file SHA-256 manifest is recorded in results.json; verify it against the
checkout before attributing these measurements to another revision.

Source baseline: `85d0ef16f1481f8edefe9481047f7666497c8ee4` plus the code changes accompanying this report.
The baseline hash alone does not identify the measured working tree. Results are
machine-specific observations; reruns need not have identical timings.
