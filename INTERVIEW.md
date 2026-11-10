# LogFlow interview walkthrough

The maintained source-backed walkthrough is [interviewprep.md](interviewprep.md).
Read [architecture.md](architecture.md) for the active ordered event-time engine
and [decision notes](explaining-each-decision-taken-over-others.md) for tradeoffs.

New source: internal/analysis/sensitivity.go computes the evidence-local grid;
internal/analysis/replay.go defines reproducible synthetic blind-spot scenarios.
The interview guide includes the seventh-event counterexample and defensible
resume wording. Start there before demonstrating the feature.
