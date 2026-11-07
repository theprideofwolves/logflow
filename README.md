# LogFlow

A focused Go log investigation app: repeated failed logins in an event-time
window, contributing-event evidence, and analyst-approved Jira investigation.

## Two focused capabilities
1. Detect a precise repeated-failure signal in an inclusive configurable window.
2. Show the trigger's contributing records and export/handoff the investigation.

Launch and run the offline synthetic demo, or upload JSONL. Each line requires
timestamp, level, and message; source_ip is required for per-IP detection.
The only rule counts exact message "Failed login attempt" with a valid source IP.
An ERROR level alone does not count. Default: 6 failures within 10 minutes,
inclusive boundaries. IPv4/IPv6 addresses are normalized.

```
logflow -window 10m -threshold 6 -workers 4 input.jsonl
logflow -window 10m -threshold 6 -
```
No arguments opens the UI; stdin now requires `-`. CLI stdout is a JSON report.
Exit 0: complete processing (alerts may exist). Exit 2: invalid configuration,
incomplete input, or tool failure. This is not a CI pass/fail security verdict.

Input must be in nondecreasing timestamp order. Out-of-order records are rejected
and completeness is false; sort upstream and rerun. Parsing is concurrent in
bounded batches; reduction preserves source order. No processing-time windows.
One alert per active threshold episode, rearmed when the current count falls
below threshold. Evidence contains the trigger window, not all later failures.
Each input line is an observation: source duplicate records must be normalized
before import; the tool has no universally reliable event-ID scheme.

Defaults: 4 parser workers, 10000 active IPs, 1000 alerts, 1 MiB line limit,
128 MiB UI request limit, 1000 stored rejection details. Parsing batches are
bounded by 256 lines and about 4 MiB of raw data. Limits/cancellation/read failures
produce partial reports rather than clean success. Accepted+rejected equals
processed lines; there may be an unread suffix. Prefix-only hashes are labeled.
Only recognized failures are retained for alert evidence, capped by threshold.

Latest report persists across restarts; export before replacing it. Stored
evidence includes contributing IPs/times and the fixed event message, not full
input logs. Jira disclosure includes IP, counts/time, input hash and completeness.
Alerts are signals for investigation, not confirmed attacks. Incomplete reports
can be handed off explicitly with their completeness limitation included.

## Installation and privacy
Download the platform ZIP from GitHub Releases and extract it. On Windows,
double-click the executable. On Linux, run the binary; if no desktop opener is
installed, copy the printed localhost URL into a browser. No development tools,
Docker, database server, or paid API is needed for the offline workflow.
The executable is unsigned: OS warnings are possible. Close the application with
Ctrl+C in its console. Closing the browser does not stop it.

The server binds to an ephemeral localhost port and uses a private per-launch
token, exact Host and Origin checks, CSP, HTML escaping and request limits.
Single-user OS trust boundary: no roles or enterprise multi-tenancy. Local
privileged processes can read memory and data. Do not expose it to a network.
No telemetry, background uploads, automatic remediation or AI dependency.

Data defaults to the OS user configuration directory / application name.
Set PORTFOLIO_DATA_DIR to change its parent; PORTFOLIO_NO_BROWSER=1 suppresses
browser opening. Only one launcher may use a data directory. After a crash,
confirm the process has stopped before removing application.lock.
Stop the app before backup/restore. Replace executables to upgrade; user data
stays separate. Uninstall by deleting extracted binaries; delete the data
directory separately only if you intend to remove history.

## Release verification
CI runs unit/integration tests, race detection, vet, and builds on Windows/Linux.
Version tags or an explicit [release] commit on main publish the VERSION-selected portable ZIPs and SHA256 checksums after tests pass.
Dependency metadata is included using go version -m; it is not a full license
inventory or a signed supply-chain attestation. No unsupported scale or
enterprise-certification claim is made.

## Jira setup and delivery semantics
Use Jira Cloud's HTTPS origin, an authorized account email/API token, project
key, and a supported issue type (Task by default). Browse/Create/Comment
permissions are required as appropriate. Projects with custom mandatory fields
may reject creation: configure a suitable type or explicitly link an existing
issue. The link action validates access to the issue, not the human's choice.
Credentials are used only for the request and never saved in ticket history.
No source snippets or raw credentials are sent. Review the disclosure before send.

The app persists a create intent before contacting Jira. A missing/uncertain
response blocks repeat creates: search Jira for the shown correlation ID, then
link the existing key. Definitive 400/401/403/404/422 rejections may be corrected
and retried. No automatic retry loops. Existing mappings receive evidence
comments; an uncertain comment may have succeeded, so inspect Jira before retry.
No automatic issue closure or bidirectional/background synchronization.
Mappings are scoped to origin and local finding/alert ID. Losing local history
or using a different origin breaks local deduplication: link existing work first.
Live Jira permission/field compatibility requires actual authorized testing;
TLS fake-server tests are not proof of live integration.

## Developer checks
`go test -race ./...`, `go vet ./...`, `go build ./cmd/logflow`.
See architecture.md, interviewprep.md and explaining-each-decision-taken-over-others.md.

## Guided first run (v1.2.0)

On a new installation, choose **Try the guided demo** or **Use my data**.
The demo runs offline and explains its expected result. Setup checks are separate
from scans/evaluations and never create tickets. After a successful task, the
workspace shows the next step and remembers guide completion in onboarding.json.
That file contains only a version and completion flag, never credentials.
Use **Restart the guide** at any time; it does not reset existing reports.
**Continue to existing workspace** opens saved work without rerunning a task.

Jira-enabled tools offer a read-only sign-in check against /rest/api/3/myself.
This verifies authentication only, not project creation/comment permissions.
The check does not retain email/API tokens. Re-enter credentials when sending
a reviewed finding. The AWS setup check in S3 Auditor verifies account identity
only; it does not claim bucket access is verified.

CredDrift offers an explicit native folder picker, a readability check, and
transfer of the selected path into the scan form. On Linux the optional picker
uses zenity; if unavailable, paste an absolute path instead. Vanguard and LogFlow
offer downloadable example inputs. Vanguard evaluation needs no connection;
optional Slack remains a CLI feature and this guide never sends Slack messages.

A corrupt/missing guide-preference file shows the guide again without changing
application results. If progress cannot be saved, the guide says so; exports and
existing domain data are not discarded. Setup is single-user and local.
