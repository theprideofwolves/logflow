# LogFlow: interview walkthrough

## Thirty-second explanation

“LogFlow is a local Go authentication-log investigation tool. It detects repeated
failed logins using ordered event-time windows and shows the records that triggered
each alert. I extended it with evidence-only sensitivity analysis and deterministic
replays to make threshold dependence, benign alerts and evasion blind spots explicit.
It supports JSON export and reviewed Jira handoff with uncertain-delivery protection.”

Use first-person claims only for work you understand and can demonstrate. AI-assisted
development is part of this project's history; do not imply unaided authorship,
production deployment, customers, or globally novel research.

## Five-minute demonstration

1. Launch with a disposable LOGFLOW_DATA_DIR parent. Run the offline demo.
2. Show eight accepted records, one alert for 192.0.2.10 and six contributing rows.
3. Expand sensitivity. Five and six are supported at the base window; seven is
   not supported by the six retained records. Explain why that is NOT a full-file
   conclusion: the demo has eight events and would actually support seven.
4. Open the replay JSON or run `logflow -replay`. Read counts 1, 0, 0, 1:
   concentrated failures alert, slow and distributed attempts evade, benign
   stale-credential failures also alert. All expectation checks should be true.
5. Export the report. Point to Options, Complete, InputHash, Evidence and Sensitivity.
   Explain Jira is optional, mocked in tests, and blocked for demo results.

## Source-backed questions and answers

**What is the rule?** Exact message “Failed login attempt”, valid normalized IP,
by default six events in an inclusive ten-minute window. Level is required input
but ERROR alone is not a failed login. Each input line counts as one observation.
The rule detects a pattern, not malicious intent or confirmed compromise.

**Why event time?** A file replay should reflect when records occurred rather than
CPU speed. Concurrent parsing produces an ordered batch; the reducer processes
source order. Nondecreasing timestamps are required. Earlier timestamps are rejected
and mark the report incomplete rather than silently sorted.

**How does sensitivity work?** Threshold minus one/base/plus one crossed with
half/base/double window, omitting settings outside valid limits. For each cell,
a two-pointer scan finds the largest inclusive count in that alert's retained
trigger events. Supported means that subset suffices. Not supported means the
subset does not suffice; later/unretained events may still support the setting.

**Why not rerun the entire input?** That would require retaining raw logs, reopening
an available source or accepting another upload. Stdin cannot be rewound. Current
scope preserves bounded storage and makes the narrower claim explicit. A future
full-input replay would need a separate input-retention and privacy contract.

**Why not call it confidence?** No probabilistic model or empirical calibration
exists. Counts and comparisons are reproducible facts; an attack probability would
be fabricated. A stable threshold pattern can still be benign.

**Does the replay measure accuracy?** No. Four deterministic fixtures exercise
known behavior with expected counts. The benign/attack distinction is supplied by
the scenario, not inferred from identical failure records. Precision and recall
would require a representative labelled evaluation with defined units and splits.

**Which evasions does it miss?** Attempts spaced beyond the window or spread across
IPs can stay below the per-IP threshold. Shared NAT may combine benign users;
account/destination context is not correlated. Password spraying, compromise and
attack attribution are not established by the current rule.

**What are the costs?** Parsing uses bounded batches (256 lines/~4 MiB), up to 32
workers, and ordered reduction. Per-IP retained events are capped at threshold T;
active IPs and alerts are capped. Sensitivity adds at most nine O(T) scans per alert,
with T at most 100. State sweeps can scan all active IPs once per event-time second
and on capacity pressure. Do not describe total runtime as unconditionally linear:
IP sweeps and source cardinality matter. No measured throughput claim is made.

**How are partial results handled?** Rejections mark Complete false. Limits, read
errors and cancellation also explain early stopping; early-stop hashes are marked
partial-prefix. Accepted + rejected equals processed lines, not necessarily every
line in the file. Cancellation is checked between batches and cannot interrupt an
arbitrary reader blocked in Read. Incomplete replay rows do not pass expectation checks.

**What persists?** The latest report contains trigger evidence, input identity,
options and sensitivity, not all raw logs. Atomic temporary-file replacement avoids
replacing the current result if save fails. Guide progress and ticket delivery state
are separate. Old report JSON without sensitivity still loads with a reanalysis hint.
The synthetic replay does not replace the latest investigation.

**Why can Jira still be uncertain?** A remote create can succeed while its response
is lost. Durable intent is saved first; pending/uncertain state blocks blind create
retries. Explicitly search/link the existing issue. Successful mappings receive
comments; comments are not exactly-once. Origin aliases share local protection.
This is not a distributed exactly-once guarantee or proof of live Jira compatibility.

**What protects the UI?** Localhost binding, per-launch token, exact host/origin
checks, request limits, escaped templates and restrictive CSP. Single-user local OS
trust remains assumed. Theme preference is a local cookie; no script or external
asset is needed. Demo results are blocked from real Jira sends on the server.

## Code map

| Concern | Read first |
| --- | --- |
| CLI, explicit stdin and -replay | cmd/logflow/main.go |
| Ordered parser/reducer and limits | internal/analysis/engine.go |
| Sensitivity grid and evidence scope | internal/analysis/sensitivity.go |
| Synthetic cases through the actual engine | internal/analysis/replay.go |
| Boundary/scope regression tests | internal/analysis/sensitivity_test.go |
| UI, persistence, export and replay endpoint | cmd/logflow/ui.go; workspace.html |
| Durable handoff and origin normalization | internal/ticket/ticket.go |
| Token/host protections and theme | internal/desktop |
| First run and setup | internal/onboarding |
| Built-binary verification | .github/scripts/smoke.py |

## Defensible résumé wording

“Extended a Go log-investigation tool with bounded event-time detection,
contributing-event evidence and threshold/window sensitivity analysis; developed
offline replays demonstrating slow/distributed blind spots and benign alerting.”

“Validated parsing, boundary conditions, partial results and Jira delivery safety
using race-enabled tests, synthetic replay fixtures and local TLS mocks.”

Adjust verbs to your actual contribution. Do not claim SOC adoption, false-positive
reduction, production scale, calibrated confidence, or an independently validated
novel algorithm. Before interviewing, explain the seventh-event counterexample
without looking at this guide, and trace one fixture through the actual code.
