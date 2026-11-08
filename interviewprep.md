# LogFlow: interview study guide

## Explain it in 30 seconds
Describe the two capabilities in README, show one input and its actual output,
then state one limitation. Do not imply enterprise deployment or real users.

## Two-minute walkthrough
Trace launcher/CLI, validation, core logic, persistence/export and the explicit
integration. Follow the functions named in architecture.md using actual source.
Identify what is local, what is transmitted and which errors preserve old state.

## Five-minute demo
Launch the packaged binary; run the synthetic example; show the explanation and
export. Demonstrate one failure. Explain what has and has not been tested live.

## Questions to practice
1. Why this input schema, and what is rejected?
2. What is the exact boundary condition for a decision or status change?
3. How are memory/concurrency bounded? Where could performance degrade?
4. What happens after cancellation, read failure or disk-write failure?
5. Which evidence is enough to support a conclusion, and which is not?
6. What does a passing automated test establish about a live provider?
7. Why not add ServiceNow, AI, or multi-tenancy immediately?
8. How would you test migration and backup restore safely?
9. How do local authorization and credential handling work?
10. Explain a duplicate external write and why local transactions cannot prevent it alone.

## Exercises
Read the feature tests and predict a boundary-case output before running them.
Introduce a failing test, explain its cause, repair it, and demonstrate the result.
Reproduce the packaged demo on a machine without Go. Explain every limitation
listed in README rather than memorizing a marketing description.

## Honest contribution statement
I used AI-assisted development to extend an existing Go tool, reviewed the
workflow and implementation, and validated defined cases with automated tests.
Replace this with specific actions you actually performed and can demonstrate.
Documentation supports study; it cannot guarantee absence of knowledge gaps.

## Answers grounded in LogFlow

The rule is the exact message "Failed login attempt" with a valid normalized IP;
ERROR alone does not count. By default six failures in an inclusive ten-minute
event-time window trigger one investigation alert. It is not proof of compromise.

Four parser workers are the default, not a proven optimum. Ordered reduction
keeps decisions independent of parser scheduling. Out-of-order events are
rejected rather than silently sorted; the report becomes incomplete.

Per-IP evidence is capped at the configured threshold. Alerts, active IPs, line
lengths and batches have explicit bounds. Explain the limit and unread-suffix
fields instead of claiming a partial result represents the complete file.

Jira intent is stored before issue creation. An uncertain response blocks blind
recreation until the analyst finds and links the issue. This is local duplicate
protection, not an exactly-once distributed guarantee. Setup verifies identity
only; handoff needs separate project permissions. Demo alerts cannot be sent.

Trace these answers through engine_test.go, ui_test.go, ticket_test.go and the
LogFlow-only packaged smoke test. Guide completion is not a security verdict.
