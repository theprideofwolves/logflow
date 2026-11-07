package main

import (
	"encoding/json"
	"fmt"
	"github.com/theprideofwolves/logflow/internal/analysis"
	"github.com/theprideofwolves/logflow/internal/desktop"
	"github.com/theprideofwolves/logflow/internal/ticket"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type saved struct {
	Report *analysis.Report
	Demo   bool
}

func saveReport(path string, v saved) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), "report-*.tmp")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	f.Chmod(0600)
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}
func launchUI() error {
	dir, e := desktop.DataDir("LogFlow")
	if e != nil {
		return e
	}
	tickets, e := ticket.Open(filepath.Join(dir, "tickets.json"))
	if e != nil {
		return e
	}
	var initial saved
	b, e := os.ReadFile(filepath.Join(dir, "report.json"))
	if e == nil {
		if e = json.Unmarshal(b, &initial); e != nil {
			return fmt.Errorf("saved report is corrupt; restore backup before continuing")
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	return desktop.Serve("LogFlow", dir, func(token string) http.Handler { return newUI(dir, token, tickets, initial) })
}
func newUI(dir, token string, tickets *ticket.Store, current saved) http.Handler {
	var mu sync.Mutex
	tmpl := template.Must(template.New("page").Parse(strings.ReplaceAll(logflowHTML, "STYLE", desktop.Style)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		message := ""
		if r.URL.Path == "/export" {
			if r.Method != "GET" || current.Report == nil {
				http.Error(w, "Analyze input first", 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", "attachment; filename=logflow-investigation.json")
			json.NewEncoder(w).Encode(current)
			return
		}
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method == "POST" {
			if e := r.ParseMultipartForm(1 << 20); e != nil {
				http.Error(w, "Invalid or oversized request", 400)
				return
			}
			defer r.MultipartForm.RemoveAll()
			switch r.FormValue("action") {
			case "demo", "analyze":
				opts := analysis.DefaultOptions()
				var input io.Reader
				var file io.Closer
				isDemo := r.FormValue("action") == "demo"
				if isDemo {
					input = strings.NewReader(analysis.Demo())
				} else {
					var e error
					opts.Threshold, e = strconv.Atoi(r.FormValue("threshold"))
					if e != nil {
						message = "Threshold must be an integer"
						break
					}
					opts.Window, e = time.ParseDuration(r.FormValue("window"))
					if e != nil {
						message = "Window must be a duration such as 10m"
						break
					}
					f, _, e := r.FormFile("logs")
					if e != nil {
						message = "Select a JSONL file"
						break
					}
					input = f
					file = f
				}
				if file != nil {
					defer file.Close()
				}
				report, e := analysis.Process(r.Context(), input, opts)
				if e != nil {
					message = e.Error()
					break
				}
				next := saved{report, isDemo}
				if e = saveReport(filepath.Join(dir, "report.json"), next); e != nil {
					message = "Could not save analysis; previous report retained"
					break
				}
				current = next
				message = "Analysis saved. Review completeness and contributing events before handoff."
			case "ticket":
				if current.Report == nil || current.Demo {
					message = "Choose a real analysis; synthetic alerts cannot create real tickets"
					break
				}
				var alert *analysis.Alert
				for i := range current.Report.Alerts {
					if current.Report.Alerts[i].ID == r.FormValue("id") {
						alert = &current.Report.Alerts[i]
					}
				}
				if alert == nil {
					message = "Alert not found"
					break
				}
				body := fmt.Sprintf("Repeated failed-login signal, not confirmed compromise. Source IP: %s. Trigger: %d observed failures from %s to %s, inclusive window %s. Input SHA256: %s. Complete processing: %t. No raw log messages, usernames or file paths are sent. Review local contributing events and investigate before taking action.", alert.IP, alert.Count, alert.Start.Format(time.RFC3339), alert.End.Format(time.RFC3339), current.Report.Options.Window, current.Report.InputHash, current.Report.Complete)
				var e error
				message, e = tickets.Handoff(r.Context(), ticket.Config{Origin: r.FormValue("jira"), Email: r.FormValue("email"), Token: r.FormValue("secret"), Project: r.FormValue("project"), Kind: r.FormValue("kind")}, "logflow|"+alert.ID, "Investigate repeated failed-login signal", body, r.FormValue("link"))
				if e != nil {
					message = e.Error()
				}
			default:
				message = "Unknown action"
			}
		}
		tmpl.Execute(w, struct {
			Token, Message string
			Saved          saved
			Tickets        map[string]ticket.Record
		}{token, message, current, tickets.Records})
	})
}

const logflowHTML = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>LogFlow</title><style>STYLE</style><h1>LogFlow</h1><p>One focused failed-login signal. Traceable investigation evidence.</p><p class="notice">Rule: message exactly equals "Failed login attempt" and source IP is valid. ERROR level alone does not count. Input must be in nondecreasing event-time order. Out-of-order records are rejected and completeness becomes false. Window boundaries are inclusive.</p>{{if .Message}}<p role="status">{{.Message}}</p>{{end}}<section><h2>Analyze JSONL</h2><form method="post" enctype="multipart/form-data" action="/?token={{.Token}}"><label>Log file (up to 128 MiB request)<input type="file" name="logs" accept=".jsonl,.ndjson,.json"></label><label>Failure threshold<input name="threshold" type="number" min="2" max="100" value="6"></label><label>Window<input name="window" value="10m"></label><button name="action" value="analyze">Analyze and save</button><button name="action" value="demo">Run synthetic demo</button></form><p>Required fields: timestamp, level, message. Source IP is required for this rule. One JSON object per line. Each repeated input line counts as an observation; normalize source duplicates before import.</p></section>{{if .Saved.Report}}<section><h2>Investigation {{if .Saved.Demo}}(SYNTHETIC){{end}}</h2><p>Complete: {{.Saved.Report.Complete}} · Accepted: {{.Saved.Report.Accepted}} · Rejected: {{.Saved.Report.Rejected}} · Missing IP: {{.Saved.Report.Unattributed}} · Processed lines: {{.Saved.Report.Lines}}</p><p>{{.Saved.Report.Limit}}</p><small>Input identity: {{.Saved.Report.InputHash}}</small><p><a href="/export?token={{.Token}}">Export saved investigation JSON</a></p>{{range .Saved.Report.Rejections}}<p>Line {{.Line}}: {{.Reason}}</p>{{end}}<small>At most 1000 rejection details are retained; total rejected count remains accurate. Scan errors/limits can leave an unread suffix.</small></section><section><h2>Alerts and contributing events</h2>{{range .Saved.Report.Alerts}}<article><h3>{{.IP}} · {{.Count}} failures at trigger</h3><p>{{.Start}} to {{.End}}</p><table><tr><th>Line</th><th>Timestamp</th><th>Event</th></tr>{{range .Evidence}}<tr><td>{{.Line}}</td><td>{{.Timestamp}}</td><td>{{.Message}}</td></tr>{{end}}</table><small>One alert per active threshold episode; another can occur after the count falls below threshold. Evidence is the trigger window, not every subsequent event.</small><details><summary>Create reviewed Jira investigation</summary><form method="post" enctype="multipart/form-data" action="/?token={{$.Token}}"><input type="hidden" name="id" value="{{.ID}}"><p>Sends IP, count, time range, rule window, input hash and completeness only. No raw log messages, usernames or paths.</p><section><h2>Jira handoff</h2><p>Review the selected record before sending. Sends only the disclosed finding/alert summary and correlation ID. Tokens are used for this request and never saved. Creates once; linked issues receive evidence comments. No automatic ticket closure. Inspect Jira before retrying an uncertain comment.</p><label>Jira origin<input name="jira" placeholder="https://team.atlassian.net" required></label><label>Email<input name="email" type="email" required autocomplete="off"></label><label>API token<input name="secret" type="password" required autocomplete="off"></label><label>Project<input name="project" placeholder="SEC"></label><label>Issue type<input name="kind" value="Task"></label><label>Existing key (optional; validates and links instead of creating)<input name="link" placeholder="SEC-123"></label><button name="action" value="ticket">Send reviewed record / link issue</button></section></form></details></article>{{else}}<p>No alerts under this rule. This does not establish absence of threats.</p>{{end}}</section>{{end}}<section><h2>Ticket delivery records</h2>{{range $id,$v:=.Tickets}}<p><small>{{$id}}: {{$v.Key}} ({{$v.State}})</small></p>{{else}}<p>No deliveries.</p>{{end}}</section><small>Latest analysis persists locally; export before replacing it. No source logs are copied to storage. Ctrl+C in launcher stops the app.</small></html>`
