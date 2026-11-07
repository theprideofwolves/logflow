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

## Onboarding design and study notes

The internal/onboarding package owns the welcome/setup routes, escaped progress
panel, declared non-secret session fields, and a versioned completion preference.
Core handlers explicitly call Complete only after a successful domain operation.
A failing policy evaluation is still a successful task; a partial LogFlow report
is not marked complete. Guide status is not a security/compliance conclusion.

GET/sample/setup routes do not run scans or create tickets. POST setup checks
use bounded read-only requests; Jira identity is not project authorization. Native
folder selection is explicit, bounded and has a manual fallback. Preflight checks
never store tokens, and Jira checks preserve previously selected local fields.

Why not a JS tour library? Server-rendered forms and a small guide keep packaging
light, work offline and are keyboard accessible. Each independent repository
contains its own internal copy and tests; no cross-project service is required.
That trades centralized updates for simpler standalone distribution.

Practice: explain a fresh launch, a failed check, a successful demo, persistence
after restart, and why restarting the guide cannot delete assessment data. Read
onboarding_test.go and the packaged smoke script to distinguish a connection
check from an end-to-end live integration test.
