package ticket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Record struct{ Key, State string }
type Store struct {
	mu      sync.Mutex
	File    string
	Records map[string]Record
	Client  *http.Client
}
type Config struct{ Origin, Email, Token, Project, Kind string }

func Open(path string) (*Store, error) {
	s := &Store{File: path, Records: map[string]Record{}, Client: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	b, e := os.ReadFile(path)
	if errors.Is(e, os.ErrNotExist) {
		return s, nil
	}
	if e != nil {
		return nil, e
	}
	if e = json.Unmarshal(b, &s.Records); e != nil {
		return nil, errors.New("ticket history is corrupt; restore backup before continuing")
	}
	if s.Records == nil {
		s.Records = map[string]Record{}
	}
	return s, nil
}
func (s *Store) save() error {
	b, e := json.MarshalIndent(s.Records, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(s.File), "tickets-*.tmp")
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
	return os.Rename(f.Name(), s.File)
}
func Origin(raw string) (string, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" {
		return "", errors.New("enter an HTTPS Jira origin without path or embedded credentials")
	}
	return strings.TrimSuffix(raw, "/"), nil
}

var issueKey = regexp.MustCompile(`^[A-Z][A-Z0-9_]*-[0-9]+$`)

func ADF(s string) any {
	return map[string]any{"type": "doc", "version": 1, "content": []any{map[string]any{"type": "paragraph", "content": []any{map[string]string{"type": "text", "text": s}}}}}
}

type remoteError struct{ status int }

func (e remoteError) Error() string {
	return fmt.Sprintf("Jira HTTP %d: check permissions, required fields, credentials or rate limits", e.status)
}
func (s *Store) call(ctx context.Context, c Config, method, path string, body any) (string, error) {
	b, e := json.Marshal(body)
	if e != nil {
		return "", e
	}
	r, e := http.NewRequestWithContext(ctx, method, c.Origin+path, strings.NewReader(string(b)))
	if e != nil {
		return "", errors.New("invalid Jira request")
	}
	r.SetBasicAuth(c.Email, c.Token)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	res, e := s.Client.Do(r)
	if e != nil {
		return "", errors.New("Jira transport failed; remote outcome may be uncertain")
	}
	defer res.Body.Close()
	b, e = io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if e != nil {
		return "", errors.New("Jira reply incomplete; reconcile before retry")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", remoteError{res.StatusCode}
	}
	if method == "POST" && path == "/rest/api/3/issue" {
		var v struct{ Key string }
		if json.Unmarshal(b, &v) != nil || !issueKey.MatchString(v.Key) {
			return "", errors.New("Jira reply has no valid issue key; reconcile before retry")
		}
		return v.Key, nil
	}
	return "", nil
}

// Handoff never automatically closes issues. Uncertain creates are blocked until explicit linking.
func (s *Store) Handoff(ctx context.Context, c Config, id, title, body, link string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var e error
	c.Origin, e = Origin(c.Origin)
	if e != nil {
		return "", e
	}
	if c.Email == "" || c.Token == "" {
		return "", errors.New("Jira email and API token required")
	}
	id = c.Origin + "|" + id
	old := s.Records[id]
	if link != "" {
		if !issueKey.MatchString(link) {
			return "", errors.New("invalid Jira issue key")
		}
		if _, e = s.call(ctx, c, "GET", "/rest/api/3/issue/"+link, nil); e != nil {
			return "", e
		}
		s.Records[id] = Record{link, "linked"}
		if e = s.save(); e != nil {
			s.Records[id] = old
			return "", e
		}
		return "Linked " + link, nil
	}
	if old.State == "pending" || old.State == "uncertain" {
		return "", fmt.Errorf("create may already have succeeded; search Jira for correlation %s and link the existing key", id)
	}
	if old.Key != "" {
		_, e = s.call(ctx, c, "POST", "/rest/api/3/issue/"+old.Key+"/comment", map[string]any{"body": ADF(body + "\nCorrelation: " + id)})
		if e != nil {
			return "", e
		}
		return "Evidence comment added to " + old.Key + ". Status unchanged.", nil
	}
	if c.Project == "" || c.Kind == "" {
		return "", errors.New("project key and issue type required")
	}
	s.Records[id] = Record{State: "pending"}
	if e = s.save(); e != nil {
		s.Records[id] = old
		return "", errors.New("cannot persist delivery intent; no request sent")
	}
	key, e := s.call(ctx, c, "POST", "/rest/api/3/issue", map[string]any{"fields": map[string]any{"project": map[string]string{"key": c.Project}, "issuetype": map[string]string{"name": c.Kind}, "summary": title, "description": ADF(body + "\nCorrelation: " + id)}})
	if e != nil {
		state := "uncertain"
		var re remoteError
		if errors.As(e, &re) && (re.status == 400 || re.status == 401 || re.status == 403 || re.status == 404 || re.status == 422) {
			state = "rejected"
		}
		s.Records[id] = Record{State: state}
		if se := s.save(); se != nil {
			return "", errors.New("Jira outcome unresolved and delivery save failed; reconcile before retry")
		}
		return "", fmt.Errorf("%v; delivery %s. Correlation: %s", e, state, id)
	}
	s.Records[id] = Record{key, "linked"}
	if e = s.save(); e != nil {
		s.Records[id] = Record{State: "uncertain"}
		return "", fmt.Errorf("created %s but local save failed; record the key and link before retry", key)
	}
	return "Created " + key, nil
}

const Form = `<section><h2>Jira handoff</h2><p>Review the selected record before sending. Sends only the disclosed finding/alert summary and correlation ID. Tokens are used for this request and never saved. Creates once; linked issues receive evidence comments. No automatic ticket closure. Inspect Jira before retrying an uncertain comment.</p><label>Jira origin<input name="jira" placeholder="https://team.atlassian.net" required></label><label>Email<input name="email" type="email" required autocomplete="off"></label><label>API token<input name="secret" type="password" required autocomplete="off"></label><label>Project<input name="project" placeholder="SEC"></label><label>Issue type<input name="kind" value="Task"></label><label>Existing key (optional; validates and links instead of creating)<input name="link" placeholder="SEC-123"></label><button name="action" value="ticket">Send reviewed record / link issue</button></section>`
