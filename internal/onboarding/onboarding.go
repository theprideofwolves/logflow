// Package onboarding provides a local, explicit first-task guide without changing domain results.
package onboarding

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/theprideofwolves/logflow/internal/desktop"
)

type Sample struct{ Name, Content string }
type Config struct {
	Name, Token, Host, Dir, Intro, Expected, Next string
	Steps                                         []string
	Samples                                       []Sample
	DemoFields                                    map[string]string
	Multipart, Jira                               bool
	Client                                        *http.Client
}
type Wizard struct {
	mu                  sync.Mutex
	c                   Config
	started, done, demo bool
	ready               bool
	warning             string
}

func New(c Config) *Wizard {
	w := &Wizard{c: c}
	if c.Client == nil {
		w.c.Client = &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	if c.Dir != "" {
		b, e := os.ReadFile(filepath.Join(c.Dir, "onboarding.json"))
		var state struct {
			Version   int
			Completed bool
		}
		if e == nil && json.Unmarshal(b, &state) == nil && state.Version == 1 && state.Completed {
			w.started = true
			w.done = true
		}
	}
	return w
}
func (w *Wizard) Complete() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.started = true
	w.done = true
	w.ready = true
	if w.c.Dir == "" {
		return
	}
	b := []byte(`{"Version":1,"Completed":true}`)
	f, e := os.CreateTemp(w.c.Dir, "onboarding-*.tmp")
	if e == nil {
		name := f.Name()
		defer os.Remove(name)
		e = f.Chmod(0600)
		if e == nil {
			_, e = f.Write(b)
		}
		if e == nil {
			e = f.Sync()
		}
		closeErr := f.Close()
		if e == nil {
			e = closeErr
		}
		if e == nil {
			e = os.Rename(name, filepath.Join(w.c.Dir, "onboarding.json"))
		}
	}
	if e != nil {
		w.warning = "Guide progress could not be saved. Your result is still available; export it before closing."
	}
}
func (w *Wizard) Panel() template.HTML {
	w.mu.Lock()
	data := struct {
		Name, Token, Expected, Next, Warning string
		Done, Demo, Ready                    bool
	}{w.c.Name, w.c.Token, w.c.Expected, w.c.Next, w.warning, w.done, w.demo, w.ready}
	w.mu.Unlock()
	var b bytes.Buffer
	panel.Execute(&b, data)
	return template.HTML(b.String())
}

//go:embed panel.html
var panelHTML string

var panel = template.Must(template.New("panel").Parse(panelHTML))

func (w *Wizard) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(out http.ResponseWriter, r *http.Request) {
		out.Header().Set("Cache-Control", "no-store")
		out.Header().Set("Referrer-Policy", "same-origin")
		out.Header().Set("X-Content-Type-Options", "nosniff")
		out.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
		if r.URL.Query().Get("token") != w.c.Token || (w.c.Host != "" && r.Host != w.c.Host) {
			http.Error(out, "Open the private launcher URL", 403)
			return
		}
		if r.Method != "GET" && r.Method != "POST" {
			http.Error(out, "Method not allowed", 405)
			return
		}
		if r.Method == "POST" {
			if o := r.Header.Get("Origin"); o != "" && o != "http://"+r.Host {
				http.Error(out, "Invalid origin", 403)
				return
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				http.Error(out, "Cross-site request blocked", 403)
				return
			}
		}
		if desktop.HandleTheme(out, r) {
			return
		}
		path := r.URL.Path
		if path == "/onboarding/sample" {
			if r.Method != "GET" {
				http.Error(out, "GET required", 405)
				return
			}
			for _, s := range w.c.Samples {
				if s.Name == r.URL.Query().Get("name") {
					out.Header().Set("Content-Type", "text/plain; charset=utf-8")
					out.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", s.Name))
					io.WriteString(out, s.Content)
					return
				}
			}
			http.NotFound(out, r)
			return
		}
		if path == "/onboarding/start" {
			if r.Method != "POST" {
				http.Error(out, "POST required", 405)
				return
			}
			r.Body = http.MaxBytesReader(out, r.Body, 16<<10)
			if e := r.ParseForm(); e != nil {
				http.Error(out, "Invalid request", 400)
				return
			}
			if r.FormValue("mode") != "own" && r.FormValue("mode") != "resume" {
				http.Error(out, "Choose your own data or resume", 400)
				return
			}
			w.mu.Lock()
			w.started = true
			w.demo = false
			w.mu.Unlock()
			http.Redirect(out, r, "/?token="+url.QueryEscape(w.c.Token), http.StatusSeeOther)
			return
		}
		if path == "/setup/check" {
			if r.Method != "POST" {
				http.Error(out, "POST required", 405)
				return
			}
			r.Body = http.MaxBytesReader(out, r.Body, 32<<10)
			if e := r.ParseForm(); e != nil {
				http.Error(out, "Invalid or oversized setup request", 400)
				return
			}
			if r.FormValue("check") != "jira" || !w.c.Jira {
				http.Error(out, "Unknown setup action", 400)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
			defer cancel()
			message, err := CheckJira(ctx, w.c.Client, r.FormValue("jira"), r.FormValue("email"), r.FormValue("secret"))
			ok := err == nil
			if err != nil {
				message = err.Error()
			}
			w.render(out, r, true, message, ok)
			return
		}
		w.mu.Lock()
		started := w.started
		w.mu.Unlock()
		if path == "/welcome" || path == "/setup" || (path == "/" && r.Method == "GET" && !started) {
			if r.Method != "GET" {
				http.Error(out, "GET required", 405)
				return
			}
			w.render(out, r, path == "/setup", "", false)
			return
		}
		if path == "/" && r.Method == "POST" {
			w.mu.Lock()
			w.started = true
			w.ready = false
			w.demo = r.URL.Query().Get("onboarding") == "demo"
			w.mu.Unlock()
		}
		next.ServeHTTP(out, r)
	})
}
func (w *Wizard) render(out http.ResponseWriter, r *http.Request, setup bool, message string, ok bool) {
	path := "/welcome"
	if setup {
		path = "/setup"
	}
	w.mu.Lock()
	data := struct {
		C                  Config
		Setup, OK          bool
		Message            string
		Theme, Token, Path string
	}{w.c, setup, ok, message, desktop.Theme(r), w.c.Token, path}
	w.mu.Unlock()
	out.Header().Set("Content-Type", "text/html; charset=utf-8")
	welcome.Execute(out, data)
}
func CheckJira(ctx context.Context, client *http.Client, origin, email, secret string) (string, error) {
	u, e := url.Parse(strings.TrimSpace(origin))
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("Use the HTTPS Jira site origin, for example https://team.atlassian.net, without a path.")
	}
	if email == "" || secret == "" {
		return "", errors.New("Enter your Jira email and API token. Neither is saved by the setup check.")
	}
	req, e := http.NewRequestWithContext(ctx, "GET", strings.TrimSuffix(u.String(), "/")+"/rest/api/3/myself", nil)
	if e != nil {
		return "", errors.New("Invalid Jira connection settings")
	}
	req.SetBasicAuth(email, secret)
	req.Header.Set("Accept", "application/json")
	c := *client
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	res, e := c.Do(req)
	if e != nil {
		return "", errors.New("Could not reach Jira. Check the site address, network and TLS settings; no ticket was created.")
	}
	defer res.Body.Close()
	if res.StatusCode == 401 || res.StatusCode == 403 {
		return "", errors.New("Jira rejected authentication. Check the email/API token and account access; no ticket was created.")
	}
	if res.StatusCode == 429 {
		return "", errors.New("Jira is rate limiting requests. Wait and try the check again; no ticket was created.")
	}
	if res.StatusCode != 200 {
		return "", fmt.Errorf("Jira returned HTTP %d. Check the site and try again; no ticket was created.", res.StatusCode)
	}
	var v struct {
		AccountID string `json:"accountId"`
	}
	if json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&v) != nil || v.AccountID == "" {
		return "", errors.New("The site did not return a valid Jira Cloud identity. Check the Jira origin.")
	}
	return "Jira sign-in works. No ticket was created. Project creation/comment permissions are checked separately when you send a LogFlow alert. Re-enter your token at handoff; it was not saved.", nil
}

//go:embed welcome.html
var welcomeHTML string

var welcome = template.Must(template.New("welcome").Parse(desktop.ThemeControl + strings.ReplaceAll(welcomeHTML, "STYLE", desktop.Style)))
