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

### Detection and completeness

The exact message `Failed login attempt` counts only with a valid source IP.
IPv4-mapped IPv6 and equivalent IPv6 spellings share a normalized address.
The level must be present but is not the detection criterion. Missing IPs are
accepted and counted as unattributed; invalid IPs and out-of-order events are
rejected. Each line is an observation; normalize source duplicates before import.

The default rule is six events in an inclusive ten-minute event-time window.
One alert is emitted per active threshold episode; it rearms when the count
drops below threshold. Evidence contains the triggering records, not later
failures. No processing-time windows or automatic timestamp sorting are used.

Defaults: four parser workers, 10,000 active IPs, 1,000 alerts, 1 MiB line limit,
128 MiB UI request limit, and 1,000 retained rejection details. Raw parsing
batches contain at most 256 lines and approximately 4 MiB. Only recognized
failures are retained per IP, capped at the configured threshold.

Rejections, limits, cancellation, and input read failures mark the report
incomplete. Accepted + rejected equals processed lines; an unread suffix may
remain. Early-stop hashes are labeled `partial-prefix`. Cancellation is checked
between bounded batches; it cannot interrupt an arbitrary reader blocked in
`Read`. A disconnected browser may need to reopen the workspace to see the
persisted partial result. No sustained-throughput claim is made.

### Local persistence and appearance

`LOGFLOW_DATA_DIR` specifies a parent directory; the app adds `LogFlow` beneath it.
Without the override it uses the OS user configuration directory. Latest report,
ticket intent/mappings, and guide completion are separate JSON files. The report
contains trigger evidence and input identity, not a copy of the entire log file.
Export before replacing it. Stop the app before backup/restore.

Only one launcher can use a data directory. After a crash, verify the process
has stopped before removing `application.lock`. Corrupt saved reports or ticket
JSON stop startup rather than silently overwriting history. Uninstalling the
binary does not remove saved data. Release executables are unsigned.

The welcome, setup and workspace share one stylesheet and a teal accent.
Light mode uses white; dark mode uses a true-black page background. A local
HttpOnly, SameSite cookie remembers the explicit choice across navigation.
Theme changes use protected POST requests; no JavaScript or remote assets are
needed. Cookie-disabled browsers cannot retain the preference.

### Jira delivery

Optional Jira Cloud handoff sends the IP, trigger count/time range, rule window,
input hash, completeness and correlation ID. It never sends raw source logs or
saves the email/API token. Synthetic demo results are blocked from handoff.
The optional setup check only reads `/rest/api/3/myself`; it does not verify
project create/comment permissions. Custom required fields may reject creation.

A durable intent is saved before creation. Definitive HTTP 400/401/403/404/422
rejections allow a corrected retry. Lost replies, malformed successful replies,
rate limits and server failures are uncertain: repeat creation is blocked until
the user searches Jira by correlation ID and explicitly links the existing key.
Origin spelling is normalized, with compatibility lookup for older history.
Multiple conflicting origin aliases block sending for reconciliation.

Linked findings receive evidence comments. A failed comment may have succeeded;
inspect Jira before retrying. No automatic retries, issue closure, or background
synchronization. Local mappings are scoped to origin and alert identity; losing
history breaks deduplication. Link existing work first when restoring history.

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

## Evidence sensitivity and replay

Every new alert gets a bounded grid: threshold minus one/base/plus one crossed
with half/base/double the configured window. Invalid settings outside supported
bounds are omitted. Each cell computes the largest inclusive sliding-window count
in the retained trigger evidence. Supported means that subset alone satisfies
the candidate rule. Not supported means only that the subset does not suffice:
it says nothing definitive about unretained events or the rest of the file.
There is no probability, confidence score, automatic threshold tuning, or full-file
counterfactual claim. The selected rule, original alert ID, and evidence stay unchanged.

At most nine cells each scan at most 100 trigger records: O(9T) time per alert,
where T is the threshold, and constant working space beyond the output cells.
The engine already guarantees ordered, same-IP evidence; Sensitivity relies on
that internal invariant. Report JSON adds Alert.Sensitivity without changing the
existing report version or alert identity algorithm. Older reports load unchanged;
the UI explicitly offers reanalysis when the field is absent. Replay results
are separate from the latest investigation and cannot create Jira tickets.

`logflow -replay` runs four fixed synthetic scenarios through the same Process
function and defaults as real analysis. Concentrated failures and a deliberately
indistinguishable stale-credential scenario alert; spaced-out and per-IP-distributed
attempts do not. Labels are scenario assumptions, not intent inferred by the
engine. Expected-alert agreement tests a known contract, not precision/recall.
GET /replay exposes the same results behind normal token/host/origin protections.
Cancellation leaves replay rows incomplete and expectation agreement false.

The project still consumes normalized ordered JSONL; native SSH/Windows ingestion,
real-world labelled evaluations and throughput claims remain future work.
