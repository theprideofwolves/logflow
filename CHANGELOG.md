# Unreleased

- Add bounded, evidence-only threshold/window sensitivity to new alerts and JSON export.
- Add offline concentrated/slow/distributed/benign authentication replay via CLI and UI.
- Document evidence scope, benign alerts, evasion blind spots and interview tradeoffs.
- Retain backward compatibility for saved reports without sensitivity fields.

# v1.2.1

Removed unrelated onboarding text and unused folder/setup code. Made CI smoke
checks LogFlow-only. Corrected architecture and interview notes to the active
time-window detector. Renamed launcher environment overrides to LOGFLOW_*.
Removed obsolete, unused stream-dispatcher code and unreferenced legacy assets.

# v1.2.0

Guided first run, offline failed-login demo, JSONL example download, own-log setup,
optional read-only Jira identity check, remembered completion and safe guide restart.

# v1.1.0

Local UI, event-time detection, bounded input processing, evidence export,
explicit Jira handoff, portable releases and failure-case tests.
