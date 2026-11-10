# LogFlow: accepted decisions

1. One explicit failed-login predicate: generic ERROR is not authentication failure.
2. Ordered reduction after bounded parallel parsing: concurrency cannot change event-time semantics.
3. Reject late events rather than buffer indefinitely: clear supported ordering contract and incomplete flag.
4. Inclusive window, alert at >= threshold: transparent boundary; one alert per active episode.
5. Bound state and report limitations: avoid claiming arbitrary input size with unlimited memory.
6. Retain latest analysis only: simple local product; user exports reports before replacing them.
7. Jira durable intent over blind retries: a successful remote write can lose its response.
Manual reconciliation handles ambiguity; evidence comments are not exactly-once.

Common choice: standard-library embedded UI over a SPA/framework. Fewer runtime
dependencies and straightforward packaging; trades rich asynchronous interactions
for full-page forms and serialized actions. Revisit after measured user need.

Alternatives deferred: ServiceNow adapters, cloud hosting, user accounts, automatic
remediation, generalized rule engines and AI. They expand operational scope without
improving the focused investigation workflow. Revisit only with a specific requirement.

Performance tradeoffs: bounds are intentional; startup and real-workload latency
must be measured before making claims. Source and tests are authoritative.

## LogFlow onboarding decisions

Use server-rendered forms rather than a JavaScript tour dependency: the binary
stays standalone and the synthetic log demo runs offline. Setup offers a JSONL
sample and optional Jira identity check, matching the actual input and adapter.
No directory scan or generic connection-check plugin is needed for log uploads.

Keep the small completion preference separate from report/ticket state so that
restarting the guide cannot delete an investigation. Mark completion only after
complete analysis, because input rejection/limits must remain visible.

Use Referrer-Policy same-origin so normal local browser form posts preserve a
valid Origin while foreign and opaque origins remain rejected by request guards.

## Sensitivity and evasion decisions

Use retained evidence rather than buffering or reopening the original log: bounded
memory, stdin compatibility and no additional raw-log retention. The cost is a
strictly local claim. A higher-threshold unsupported cell must never be presented
as proof that the full input would stop alerting.

Use a fixed neighborhood rather than an unrestricted rule search: at most nine
comparisons, no tuning to a desired result. Omit candidates outside valid bounds.
Use inclusive boundaries identical to Process. No synthetic confidence percentage.

Use deterministic synthetic replays through Process rather than a separate toy
detector. Include a benign alerting scenario to expose the predicate's inability
to establish intent. Do not claim attack-classification accuracy from four fixtures.
Keep replay read-only and separate from persistence and Jira handoff.
