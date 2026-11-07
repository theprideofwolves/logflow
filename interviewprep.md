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
