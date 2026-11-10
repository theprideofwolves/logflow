# Testing LogFlow

Automated tests use synthetic logs, temporary directories and local TLS Jira
mocks. They do not establish live Jira compatibility or exhaustive edge-case
coverage. No credentials, paid APIs or cloud resources are needed.

## Automated checks

Run from the repository with Go 1.25.5+ and Python 3. Race testing also needs a
supported C compiler.

```sh
go test -race -count=1 ./...
go vet ./...
go build -o logflow ./cmd/logflow
python .github/scripts/smoke.py ./logflow
```

On Windows, build `logflow.exe` and pass `./logflow.exe` to the smoke script.
The script creates its own temporary data parent and disables browser opening.
Go tests use `t.TempDir()`; they do not read the normal application data directory.

Optional bounded fuzz checks:

```sh
go test ./internal/parser -run '^$' -fuzz '^FuzzParseLine$' -fuzztime 15s -parallel 2
go test ./internal/analysis -run '^$' -fuzz '^FuzzProcessReportInvariants$' -fuzztime 15s -parallel 2
```

Covered scenarios include:

- Inclusive/exclusive window boundaries, threshold/rearm, exact-message matching,
  IPv4/IPv6 normalization, source separation and contributing evidence.
- Malformed/missing/wrong-type fields, invalid IPs, missing IP attribution,
  out-of-order records, rejection accounting, read failures, limits and cancellation.
- Invalid CLI arguments and exit codes; own-file upload, JSON export, failed-save
  preservation, concurrent requests, HTML escaping and guide completion/restart.
- Light/dark preference across pages, token/origin checks, unsafe redirects,
  oversized theme requests and demo handoff blocking.
- Mocked Jira rejection/retry, lost or malformed replies, persisted pending intent,
  explicit linking, origin aliases, concurrent duplicate-create protection and
  credential non-retention.
- Built-executable CLI, welcome/setup/sample download, themes, demo/upload/export,
  single-instance lock and report persistence across process restart.

## Quick manual checklist (about 5 minutes)

Use a disposable data parent. On Windows, after building `logflow.exe`:

```powershell
$env:LOGFLOW_DATA_DIR = Join-Path $env:TEMP ('logflow-review-' + [guid]::NewGuid())
Remove-Item Env:LOGFLOW_NO_BROWSER -ErrorAction SilentlyContinue
.\logflow.exe
```

Keep this terminal open. Reuse the same terminal/environment for the restart check.
The app adds `LogFlow` beneath that temporary parent; your regular data stays separate.

- [ ] **Appearance:** welcome is white with teal accents. Switch to dark mode:
  page background is black, accents remain teal, and text/inputs are readable.
  Visit setup and return; refresh. The chosen theme stays selected.
- [ ] **Demo:** choose **Try the guided demo**. Confirm `Complete: true`, eight
  accepted records, one alert for `192.0.2.10` and six contributing rows. There
  should be no Jira credential/send form for the demo.
- [ ] **Export:** download the investigation JSON and open it. Confirm `Demo: true`,
  one alert and six evidence records. The file should contain no credentials.
- [ ] **Own-file flow:** open setup, download `example-logs.jsonl`, open the
  workspace, select that downloaded file and analyze with defaults. Confirm the
  same counts, now without the synthetic label. Leave Jira untouched.
- [ ] **Persistence:** restart the guide and continue to the workspace; the result
  remains. Press Ctrl+C in the terminal, relaunch `logflow.exe` there and confirm
  the latest result is still present.
- [ ] **Usability:** at a narrow window and 200% browser zoom, check the forms and
  evidence table remain usable. Tab through controls and verify visible focus.

Live Jira permissions/custom fields, OS download warnings, screen-reader behavior
and sustained workload measurement are separate, unverified checks. No live Jira
request is needed for this checklist. Closing the browser does not stop the app;
use Ctrl+C when finished.

## Sensitivity and replay checks

Automated tests check inclusive sensitivity counts, parameter-boundary omissions,
empty evidence, cancellation, exported fields and replay isolation. A regression
explicitly adds a seventh input event: the full input supports threshold seven,
while the six retained trigger events do not. This protects the scope distinction.

Quick manual additions:
- [ ] Run the demo and expand **How sensitive is this finding?** At the default
  ten-minute window, threshold five/six are supported; seven is not supported by
  the six retained events. Read the scope warning; do not infer full-input absence.
- [ ] Open **Run synthetic replay (JSON)**. Expect four complete rows with
  MatchesExpectation true, observed alert counts 1, 0, 0, 1. Return to the workspace
  and confirm the saved investigation is unchanged.
- [ ] Export JSON and confirm each new alert includes Sensitivity. Older saved
  reports without it should still open with an explicit reanalysis message.

CLI: `logflow -replay` emits the same four rows without reading a file or touching
application data. Combining -replay with input or analysis options is rejected.

## Reproduce the recorded results

Run `python tools/verify.py --date YYYY-MM-DD` from a clean checkout with Go,
a race-compatible C compiler and Python 3.10+. The runner uses temporary application
data, disables automatic browser launch, runs race tests/coverage, vet, native build,
packaged smoke, four replays, bounded fuzzing, benchmarks and a Linux cross-build.
It writes docs/results/results.json, benchmarks.txt and README.md. Failed checks
stop the run; do not treat a previous report as evidence for a failed new run.

Benchmarks are three repeated samples of three analyses per case. They compare
one/four workers over 10k records/one IP and 100k records/1000 IPs. All events
match the rule; exact expected counts are asserted. Allocation volume is cumulative
B/op, not peak RSS, retained state, file size or total application memory. Results
exclude fixture generation, disk I/O, JSON serialization, UI and Jira. Hardware,
Go version, GOMAXPROCS and all observed ranges are recorded; do not cherry-pick
a single fastest sample or generalize these figures to production performance.
