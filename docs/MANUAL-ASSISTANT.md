# Manual companion checks

Status: **Not yet observed.** The operator will perform these checks. Do not mark
items passed just because code or regression tests exist. Use synthetic logs first,
then approved lab-generated data with documented normalization.

## Start safely

Build the current source with `go build -o logflow.exe ./cmd/logflow`, then:

```powershell
$env:LOGFLOW_DATA_DIR = Join-Path $env:TEMP ('logflow-manual-' + [guid]::NewGuid())
Remove-Item Env:LOGFLOW_NO_BROWSER -ErrorAction SilentlyContinue
.\logflow.exe
```

Reuse this terminal for restart checks. Never photograph the private launcher URL,
API token, environment dump or personal account details. Use synthetic/private lab
IPs for a public video and hide the address bar.

| Check | Expected observation | Result / evidence |
| --- | --- | --- |
| Demo analysis | One alert, six contributing records, complete input | Pending |
| Offline draft, no provider installed | Facts, limitations, four guidance sections; no payment/account prompt | Pending |
| Citations | Finding/report/line references resolve to the selected snapshot; section IDs/version visible | Pending |
| Draft export and restart | Export includes trace/corpus hash; latest draft reopens after restart | Pending |
| New analysis | Previous draft is not presented as describing the new report | Pending |
| Incomplete input | Upload sample plus malformed line; draft clearly states incomplete evidence | Pending |
| Missing provider model/key | Visible warning and useful offline fallback; no clean provider-success claim | Pending |
| Local model, optional | Set your installed model; inspect selected sections and trace; no writer call | Pending |
| OpenAI without consent | Request refused before external processing | Pending |
| OpenAI with your own key, optional paid check | Context disclosure accepted; bounded plan or visible fallback; key absent from export | Pending |
| MCP with trusted local client | Exactly three tools; selected evidence readable; unknown section/write tool fails | Pending |
| Jira preview on own-file analysis | No issue before approval; inspect exact destination and disclosure | Pending |
| Decline Jira | Leave preview; no request or issue | Pending |
| Approve in disposable Jira project | Exactly intended create/comment/link; delivery state visible | Pending |
| Reuse, stale report, restart or expired preview | No send; new preview required | Pending |
| Existing uncertain delivery | Reconcile/link; no blind duplicate create | Pending; prefer local mocks |
| Audit files | Draft/trace and action records exist without credentials | Pending |
| Light/dark and 200% zoom | Drafts and preview are readable, keyboard focus visible | Pending |

Do not deliberately interrupt a live Jira create to manufacture uncertainty. Use
mocked recovery evidence and label it as mocked. Live Jira checks need a disposable
project and your explicit approval, not workplace credentials.

## Record results

For each observation record: source commit, OS/build, input hash and provenance,
mode/provider/model (no key), steps, expected/actual behavior, pass/fail/unverified,
and screenshot or video timestamp. Label synthetic, sanitized lab, mocked and live
integration evidence separately. Do not call a fixture agreement percentage field
accuracy. A locally running model is not automatically private if its runtime uses
a cloud model.

Suggested video: demo -> evidence -> offline draft/citations -> optional local-model
selection -> Jira preview -> separately approved disposable issue. Keep it short;
show a failure/fallback and limitation rather than only the happy path. Review every
frame for secrets before uploading. Do not commit raw workplace logs or local audit
files. A screenshot alone does not establish recovery, injection resistance or
permission enforcement.
