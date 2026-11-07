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
improving the two selected workflows. Revisit only with a specific requirement.

Performance tradeoffs: bounds are intentional; startup and real-workload latency
must be measured before making claims. Source and tests are authoritative.
