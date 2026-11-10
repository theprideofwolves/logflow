# Local unreleased changes

New investigations include evidence-only threshold/window sensitivity. Offline
replay demonstrates slow/distributed blind spots and a benign alerting pattern.
These are bounded evidence comparisons and synthetic behavior checks, not
confidence scores, full-file reruns or measured field detection accuracy.
Existing reports remain readable. No live Jira validation is claimed.

# LogFlow v1.2.1

Portable single-user log investigation for Windows/Linux: ordered JSONL analysis,
time-window failed-login alerts, contributing-event evidence, JSON export, and
optional analyst-approved Jira Cloud handoff.

This patch removes unrelated onboarding documentation and unused setup paths,
specializes the packaged smoke test to LogFlow, and replaces stale interview
notes with the implemented event-time architecture. Launcher overrides are now
LOGFLOW_DATA_DIR (parent directory) and LOGFLOW_NO_BROWSER. Default user data
locations and saved report/ticket formats are unchanged.

Automated gates: race-enabled Go tests, vet, native Windows/Linux builds, and
packaged launch/setup/demo/evidence/export/restart smoke checks. Jira contract
tests use TLS fake servers. Live Jira project permissions/custom fields, a clean
OS interactive launch, accessibility, and sustained workload measurement still
need separate validation. Passing tests do not establish enterprise deployment.

Packages are unsigned and include dependency build metadata and ZIP checksums.
No account, paid API, database server, or developer tools are needed offline.
