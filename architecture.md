# LogFlow: implemented architecture

Browser/CLI -> bounded input reader/hash -> fixed parser workers per bounded
batch -> ordered event-time reducer -> bounded per-IP windows -> coalesced alerts
with contributing line/time evidence -> local JSON and explicit Jira handoff.
The active CLI/UI uses internal/analysis. The older run-wide detector is removed.
Latest report writes atomically; ticket intent has separate durable state.
An input hash is partial-prefix labeled when processing stops early. Wrong-order
events are rejected transparently, not silently reordered or processed by arrival.

## Operational boundaries
Private single-user tool. Browser access requires the launcher token. No remote
deployment, authentication roles, telemetry, continuous watch, or automatic
containment. Data corruption stops load instead of silently overwriting history.
Backups and third-party credentials remain the user's responsibility.

## Tests and evidence
Feature tests are alongside implementation, local-request tests in
internal/desktop, and integration contract tests beside the adapters. CI verifies
Windows/Linux builds and race checks. Mocks verify defined contracts; live
permissions, provider behavior and clean-machine interactive UI require manual
validation. Release packaging does not establish enterprise production readiness.

## Source map and onboarding

- cmd/logflow/main.go selects CLI or the local UI and validates flags.
- internal/parser/json_parser.go validates each JSONL record.
- internal/analysis/engine.go parses bounded batches concurrently, then reduces
  results in source order into event-time windows and alert evidence.
- cmd/logflow/ui.go owns upload/demo, latest-report persistence/export and the
  reviewed Jira handoff. internal/ticket owns durable Jira intent/mapping state.
- internal/desktop owns localhost launch, token/host/origin protection and locking.
- internal/onboarding owns the welcome/setup routes, JSONL sample download,
  optional read-only Jira identity check and completion preference.

Setup never analyzes logs or sends tickets. Only a complete analysis calls
Complete; rejected/partial input does not mark guide completion. Guide progress
is independent of report.json and tickets.json. No setup credentials are stored.
The packaged smoke test launches the actual executable, inspects the LogFlow
demo evidence/export and confirms report persistence after process restart.
