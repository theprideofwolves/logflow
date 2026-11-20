package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/theprideofwolves/logflow/internal/analysis"
	"github.com/theprideofwolves/logflow/internal/assistant"
	"github.com/theprideofwolves/logflow/internal/desktop"
	"github.com/theprideofwolves/logflow/internal/onboarding"
	"github.com/theprideofwolves/logflow/internal/ticket"
)

type saved struct {
	Report    *analysis.Report
	Demo      bool
	Assistant *assistant.Draft `json:",omitempty"`
}

type jiraPreview struct {
	Nonce, Snapshot, AlertID, Body, Link string
	Config                               ticket.Config
	Expires                              time.Time
}

func reportIdentity(r *analysis.Report) string {
	b, _ := json.Marshal(r)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
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
	guide := onboarding.New(onboarding.Config{Name: "LogFlow", Token: token, Dir: dir, Jira: true, Multipart: true, Intro: "Turn repeated failed logins into a traceable investigation.", Expected: "Expect one alert for 192.0.2.10 with six contributing failed-login events. These are synthetic records, not a confirmed attack.", Next: "Open the contributing events, export the investigation, then upload your own ordered JSONL file. Jira handoff is optional.", Steps: []string{"No account is needed for local analysis. Download the example to see one JSON object per line.", "Your file needs timestamp, level and message; source_ip is needed for this rule. Sort by nondecreasing timestamp before upload.", "Open the workspace, select a file and leave the default six failures / ten-minute window for your first run.", "Review completeness and rejected records before interpreting the alerts. Jira is optional."}, DemoFields: map[string]string{"action": "demo"}, Samples: []onboarding.Sample{{Name: "example-logs.jsonl", Content: analysis.Demo()}}})
	var mu sync.Mutex
	pending := map[string]jiraPreview{}
	tmpl := template.Must(template.New("page").Funcs(template.FuncMap{"onboarding": guide.Panel}).Parse(desktop.ThemeControl + strings.ReplaceAll(logflowHTML, "STYLE", desktop.Style)))
	return guide.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		message := ""
		var preview *jiraPreview
		for k, v := range pending {
			if time.Now().After(v.Expires) {
				delete(pending, k)
			}
		}
		if r.URL.Path == "/assistant" {
			if r.Method != "GET" || current.Assistant == nil {
				http.Error(w, "Generate an investigation draft first", 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", "attachment; filename=logflow-draft.json")
			json.NewEncoder(w).Encode(current.Assistant)
			return
		}
		if r.URL.Path == "/replay" {
			if r.Method != "GET" {
				http.Error(w, "GET required", 405)
				return
			}
			results, err := analysis.Replay(r.Context())
			if err != nil {
				http.Error(w, "Replay failed", 500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)
			return
		}
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
				next := saved{Report: report, Demo: isDemo}
				if e = saveReport(filepath.Join(dir, "report.json"), next); e != nil {
					message = "Could not save analysis; previous report retained"
					break
				}
				current = next
				if report.Complete {
					guide.Complete()
				}
				message = "Analysis saved. Next: review completeness, open the contributing events, and export the investigation. Jira is optional."
			case "draft":
				if current.Report == nil {
					message = "Analyze input first"
					break
				}
				provider := r.FormValue("provider")
				if provider != "offline" && provider != "ollama" && provider != "openai" {
					message = "Choose a supported draft mode"
					break
				}
				if provider == "openai" && r.FormValue("allow_remote") != "yes" {
					message = "Confirm the external-provider disclosure before sending context"
					break
				}
				session, err := assistant.Open(current.Report, r.FormValue("id"))
				if err != nil {
					message = err.Error()
					break
				}
				draft := session.Build(r.Context(), provider)
				if err = assistant.Persist(dir, draft); err != nil {
					message = err.Error()
					break
				}
				next := current
				next.Assistant = &draft
				if err = saveReport(filepath.Join(dir, "report.json"), next); err != nil {
					message = "Draft record saved, but latest report could not be updated"
					break
				}
				current = next
				message = "Investigation draft saved locally. Review citations and limitations. No Jira action was performed."
			case "approve_ticket":
				nonce := r.FormValue("approval")
				p, ok := pending[nonce]
				if !ok || r.FormValue("confirm_send") != "yes" {
					message = "A valid preview and explicit approval are required"
					break
				}
				delete(pending, nonce)
				if current.Report == nil || current.Demo || p.Snapshot != reportIdentity(current.Report) || time.Now().After(p.Expires) {
					message = "Preview expired or investigation changed. Prepare a new preview."
					break
				}
				if r.FormValue("secret") == "" {
					message = "Re-enter your Jira API token and prepare a new preview"
					break
				}
				approvalRecord, _ := json.Marshal(p)
				if err := assistant.RecordAction(dir, "approved", p.Nonce, p.Snapshot, string(approvalRecord)); err != nil {
					message = err.Error()
					break
				}
				p.Config.Token = r.FormValue("secret")
				var err error
				message, err = tickets.Handoff(r.Context(), p.Config, "logflow|"+p.AlertID, "Investigate repeated failed-login signal", p.Body, p.Link)
				if err != nil {
					message = err.Error()
				}
				if auditErr := assistant.RecordAction(dir, "handoff finished; inspect delivery state", p.Nonce, p.Snapshot, message); auditErr != nil {
					message += ". Outcome audit failed; inspect ticket delivery records before retrying."
				}
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
				if len(r.FormValue("jira")) > 2048 {
					message = "Jira origin exceeds limit"
					break
				}
				origin, err := ticket.Origin(r.FormValue("jira"))
				if err != nil {
					message = err.Error()
					break
				}
				if len(pending) >= 20 {
					message = "Too many pending previews. Wait five minutes before preparing another."
					break
				}
				config := ticket.Config{Origin: origin, Email: r.FormValue("email"), Project: r.FormValue("project"), Kind: r.FormValue("kind")}
				if config.Email == "" || len(config.Email) > 254 || len(config.Project) > 100 || len(config.Kind) > 100 || len(r.FormValue("link")) > 100 {
					message = "Invalid Jira handoff fields"
					break
				}
				p := jiraPreview{Nonce: desktop.Token(), Snapshot: reportIdentity(current.Report), AlertID: alert.ID, Body: body, Link: r.FormValue("link"), Config: config, Expires: time.Now().Add(5 * time.Minute)}
				pending[p.Nonce] = p
				preview = &p
				message = "Review the exact disclosure. Nothing has been sent. Approval expires in five minutes."
			default:
				message = "Unknown action"
			}
		}
		tmpl.Execute(w, struct {
			Token, Message string
			Theme, Path    string
			Saved          saved
			Tickets        map[string]ticket.Record
			Preview        *jiraPreview
		}{token, message, desktop.Theme(r), "/", current, tickets.Records, preview})
	}))
}

//go:embed workspace.html
var logflowHTML string
