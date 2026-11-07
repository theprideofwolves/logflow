# LogFlow: release scope and validation

Portable single-user Windows/Linux release. See README for the two supported
workflows, setup, data storage and important limitations.

Automated gates: go test -race, go vet, Windows/Linux compilation, and package
smoke checks. Feature tests cover documented state transitions, malformed input,
boundary conditions, caps, and external failure contracts. No promise of every
conceivable edge case.

Manual validation: live Slack/Jira permissions and delivery; human accessibility
review; interactive first launch on a clean OS; long-running resource measurement;
real user pilot. These remain distinct from unit tests and HTTP smoke checks.

Unsigned portable packages include dependency build metadata, not a full SBOM.
Custom required Jira fields, provider credential vaulting, remote/shared deployment,
automatic remediation and ServiceNow are outside this focused release.

## v1.2.0 onboarding checks
Welcome/own-data choice, offline first task, expected-result explanation, next-step
panel, export, restart safety and remembered completion are exercised by the
packaged smoke test on Windows/Linux. Unit tests cover read-only Jira outcomes,
credential non-retention, field preservation, invalid setup requests, missing or
corrupt preferences, folder checks and mocked native selection. A real native
dialog and live AWS/Jira credentials still require manual validation.
