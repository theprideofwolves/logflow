# Investigation companion

LogFlow can draft an investigation without an account, model or paid API.
The detector remains deterministic. The companion is a read-only aid, not an
incident verdict or autonomous SOC agent.

## Choose a mode

| Mode | What it does | Requirements |
| --- | --- | --- |
| Offline (default) | Builds facts, evidence references, limitations and four cited project runbook sections | LogFlow only; no network call |
| Ollama (optional) | A locally configured model selects up to four approved read-only calls and relevant guidance | Your installed runtime and model; hardware/model licence requirements vary |
| OpenAI (optional) | The same bounded selection using your own API account | Your key, explicit model, network access and per-request consent; API charges apply |

Model output cannot add factual claims, free-text conclusions, new runbooks or
write actions. Invalid plans, forbidden tools, truncation and provider failures
retain the offline draft with a visible warning. This is a single bounded planning
request, not an open-ended reasoning loop. It does not browse the web or execute
commands. Default output remains useful if a model is absent.

The runbooks in internal/assistant/runbooks.json are project-authored guidance
with stable IDs, versions and source references. They are not approved enterprise
runbooks. Their exact content is included in tool traces and identified by a corpus
hash. Retrieval is an allowlisted section lookup, not semantic/vector search or a
claim of general-purpose RAG. Changes require code review and verification.

## Browser workflow

1. Build the current source and launch with a disposable LOGFLOW_DATA_DIR parent.
2. Run the demo or analyze ordered JSONL.
3. Open an alert, expand **Draft an investigation**, leave **Offline** selected,
   then generate the draft.
4. Review facts, resolve the finding/report/line references, and read limitations.
5. Read the cited runbook sections and export the draft/tool trace JSON.
6. A model can optionally select guidance. It cannot send Jira tickets or change
   the original analysis, threshold, evidence or sensitivity result.

New drafts are stored under `LogFlow/investigations/<draft-id>.json`. The latest
report includes the latest draft. Replacing an analysis clears the latest draft,
while immutable draft files remain. Local assistant-audit.jsonl stores completed
runs, prompt/version, provider context, calls/results and status. These files
include investigation metadata and IP addresses; keep them private by default.
The audit is append-only by application convention, not tamper-proof or multi-user
attribution. The assistant audit is capped at 32 MiB; archive it before more runs.
An audit failure is visible and prevents presenting a successful saved run.

## Optional model configuration

Set configuration in the terminal that launches LogFlow. Keys stay in the local
server process, never in browser JavaScript, configuration committed to Git or
provider context. Do not send production data without permission.

For an already installed local Ollama model:

```powershell
$env:LOGFLOW_OLLAMA_MODEL = 'YOUR_INSTALLED_LOCAL_MODEL'
.\logflow.exe
```

Select Ollama in the draft form. LogFlow calls only
`http://127.0.0.1:11434/api/chat`; it does not install models or contact a remote
Ollama URL. Choose a genuinely local model. The runtime's own cloud routing,
network behavior and model licence remain your responsibility. A slow or missing
runtime falls back after the request timeout.

For OpenAI, use a separate API account/key and an API model available to that account:

```powershell
$secureKey = Read-Host 'OpenAI API key' -AsSecureString
$env:OPENAI_API_KEY = [System.Net.NetworkCredential]::new('', $secureKey).Password
$env:LOGFLOW_OPENAI_MODEL = 'YOUR_API_MODEL_ID'
.\logflow.exe
```

Select OpenAI and check the external-provider disclosure on each request. LogFlow
sends the alert ID, failure count, completeness/rejection count and bundled runbook
sections. It does not send raw input logs, IPs, timestamps, Jira credentials or the
API key as prompt context. Authentication uses the key in the HTTPS header.
Responses use `store: false`. This does not establish zero retention at the provider;
review its account/data policies. No redirects, proxy environment forwarding or
automatic retries are used. The client timeout is 30 seconds; the output request
budget is 800 tokens. Account billing and provider limits still apply.

ChatGPT Plus is not a general API key or ordinary API-billing entitlement. Do not
copy browser session cookies or subscription tokens into LogFlow. You can manually
review a sanitized draft in ChatGPT using your existing subscription, but that is
not an integrated project API workflow. No developer key is bundled with LogFlow.

- [OpenAI billing](https://help.openai.com/en/articles/9039756-managing-billing-for-chatgpt-and-the-api-platform)
- [OpenAI key safety](https://help.openai.com/en/articles/5112595-best-practices-for-api-key-safety)
- [Responses API](https://developers.openai.com/api/reference/resources/responses/methods/create)
- [Ollama chat API](https://docs.ollama.com/api/chat)

## CLI and MCP

Export an investigation first. Select an alert ID from Report.Alerts. Do not use a
launcher URL or put a provider key in command arguments.

```sh
logflow -assist investigation.json -alert ALERT_ID
logflow -assist investigation.json -alert ALERT_ID -provider ollama
logflow -assist investigation.json -alert ALERT_ID -provider openai -allow-remote
logflow -assist investigation.json -alert ALERT_ID -mcp
```

CLI drafts are emitted as JSON with their trace. Redirect stdout to a private
file if you want a record; the CLI does not silently write normal application data.
Provider fallback emits an offline draft with a warning and exit code 2.

The MCP server uses the official Go SDK over local stdio. A client must explicitly
launch it with an operator-selected report and alert. Only three tools exist:

- get_finding: selected alert, configuration, completeness and input identity.
- get_evidence: that alert's contributing events.
- get_runbook_section: an approved bundled section ID.

Pass an object with an `id` field. There is no arbitrary path, URL, command or
Jira tool. MCP read events stream as JSON on stderr; stdout is reserved for the
protocol. Configure a client that supports local stdio using this pattern:

```json
{"mcpServers":{"logflow":{"command":"/ABSOLUTE/PATH/TO/logflow","args":["-assist","/ABSOLUTE/PATH/TO/investigation.json","-alert","ALERT_ID","-mcp"]}}}
```

Use a trusted client. Once a client reads evidence, its own provider, disclosure
and retention policies apply. Local stdio is not a multi-user authorization system.
Do not assume the ChatGPT website can launch a local stdio server directly.

## Separate Jira approval

The companion never holds Jira credentials or exposes a writer tool. In a non-demo
analysis, prepare a Jira handoff from the existing alert form. Review the destination,
account, project/type, body and correlation ID before entering the token and approving.
A server-held preview binds those values to the report snapshot, expires after five
minutes and is consumed once. Changed reports, missing confirmation, invented
previews and reused approvals cannot send. Pending previews do not survive restart;
prepare another one. At most 20 unexpired previews are retained.

Approval is recorded before delivery in handoff-audit.jsonl; audit failure blocks
sending. Existing durable Jira intent still handles uncertain replies. Definitive
rejections require a corrected new preview; uncertain creates require reconciliation
and explicit linking. A failed outcome audit is displayed even if Jira accepted the
request. Never blindly retry based on a missing UI response. No containment action,
provider key rotation or automatic issue closure exists.

## Public use

Other users run their own local copy, with their own data, optional model and keys.
You do not operate a hosted service or pay for their requests. Do not expose the
localhost UI as a public/shared server. Multi-user access, tenancy and centralized
secret management are outside this version's scope.

## Verification status

Build/regression checks are separate from manual observations. No live OpenAI,
Ollama, Jira or representative field evaluation is claimed for this change.
Old performance/coverage figures refer to the historical measured revision.
Use [MANUAL-ASSISTANT.md](MANUAL-ASSISTANT.md) for the pending checks. Publish only
reviewed/redacted results after the operator actually performs them.
