package ticket

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestHandoffFailureMatrix(t *testing.T) {
	for _, tc := range []struct {
		name        string
		status      int
		body, state string
	}{{"created", 201, `{"key":"SEC-1"}`, "linked"}, {"invalid reply", 201, `{}`, "uncertain"}, {"invalid key", 201, `{"key":"SEC-1/../../"}`, "uncertain"}, {"bad request", 400, `{}`, "rejected"}, {"unauthorized", 401, `{}`, "rejected"}, {"permission", 403, `{"secret":"must not echo"}`, "rejected"}, {"not found", 404, `{}`, "rejected"}, {"mandatory field", 422, `{}`, "rejected"}, {"rate limit", 429, `{}`, "uncertain"}, {"server", 503, `{}`, "uncertain"}} {
		t.Run(tc.name, func(t *testing.T) {
			calls := []string{}
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.URL.Path)
				if u, p, ok := r.BasicAuth(); !ok || u != "analyst" || p != "private-token" {
					t.Error("auth")
				}
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer srv.Close()
			p := filepath.Join(t.TempDir(), "tickets.json")
			s, e := Open(p)
			if e != nil {
				t.Fatal(e)
			}
			s.Client = srv.Client()
			c := Config{srv.URL, "analyst", "private-token", "SEC", "Task"}
			_, e = s.Handoff(context.Background(), c, "finding-1", "Review", "Redacted metadata", "")
			if tc.state == "linked" && e != nil {
				t.Fatal(e)
			}
			if tc.state != "linked" && (e == nil || strings.Contains(e.Error(), "must not echo")) {
				t.Fatal("failure hidden or remote body echoed", e)
			}
			if s.Records[srv.URL+"|finding-1"].State != tc.state {
				t.Fatal(s.Records)
			}
			b, _ := os.ReadFile(p)
			if strings.Contains(string(b), "private-token") {
				t.Fatal("token persisted")
			}
			reloaded, e := Open(p)
			if e != nil {
				t.Fatal(e)
			}
			reloaded.Client = srv.Client()
			reloaded.Handoff(context.Background(), c, "finding-1", "Review", "Redacted metadata", "")
			if tc.state == "uncertain" && len(calls) != 1 {
				t.Fatal("blind duplicate create")
			}
			if tc.state == "linked" && (len(calls) != 2 || !strings.HasSuffix(calls[1], "/comment")) {
				t.Fatal(calls)
			}
			if tc.state == "rejected" && (len(calls) != 2 || calls[1] != "/rest/api/3/issue") {
				t.Fatal("definitive rejection did not allow corrected retry", calls)
			}
		})
	}
}

func TestConcurrentHandoffsCreateOnlyOnce(t *testing.T) {
	var creates, comments atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" {
			creates.Add(1)
			w.WriteHeader(201)
			io.WriteString(w, `{"key":"SEC-1"}`)
		} else if r.URL.Path == "/rest/api/3/issue/SEC-1/comment" {
			comments.Add(1)
			w.WriteHeader(201)
		} else {
			t.Error("unexpected path", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	s, err := Open(filepath.Join(t.TempDir(), "tickets.json"))
	if err != nil {
		t.Fatal(err)
	}
	s.Client = srv.Client()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Handoff(context.Background(), Config{srv.URL, "u", "p", "P", "Task"}, "finding", "title", "evidence", "")
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if creates.Load() != 1 || comments.Load() != 7 {
		t.Fatal(creates.Load(), comments.Load())
	}
}

func TestPendingIntentReconcilesByExplicitLink(t *testing.T) {
	var creates, links atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/rest/api/3/issue/SEC-9" {
			links.Add(1)
			io.WriteString(w, `{"key":"SEC-9"}`)
			return
		}
		creates.Add(1)
		w.WriteHeader(500)
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "tickets.json")
	s, _ := Open(path)
	s.Records[srv.URL+"|finding"] = Record{State: "pending"}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Client = srv.Client()
	c := Config{srv.URL, "u", "p", "P", "Task"}
	if _, err = s.Handoff(context.Background(), c, "finding", "title", "body", ""); err == nil {
		t.Fatal("pending intent retried")
	}
	if _, err = s.Handoff(context.Background(), c, "finding", "title", "body", "SEC-9"); err != nil {
		t.Fatal(err)
	}
	if creates.Load() != 0 || links.Load() != 1 || s.Records[srv.URL+"|finding"].Key != "SEC-9" {
		t.Fatal(s.Records, creates.Load(), links.Load())
	}
}
func TestLinkValidationAndRedirection(t *testing.T) {
	for _, u := range []string{"http://jira.test", "https://u:p@jira.test", "https://jira.test/path", "https://jira.test?x=1"} {
		if _, e := Origin(u); e == nil {
			t.Fatal(u)
		}
	}
	s, _ := Open(filepath.Join(t.TempDir(), "tickets.json"))
	if _, e := s.Handoff(context.Background(), Config{"https://jira.test", "u", "p", "P", "Task"}, "id", "t", "b", "SEC-1/../x"); e == nil {
		t.Fatal("unsafe key")
	}
}
func TestCorruptAndUnwritableState(t *testing.T) {
	p := filepath.Join(t.TempDir(), "tickets.json")
	os.WriteFile(p, []byte("broken"), 0600)
	if _, e := Open(p); e == nil {
		t.Fatal("corrupt history accepted")
	}
	s, _ := Open(filepath.Join(t.TempDir(), "new.json"))
	s.File = filepath.Join(t.TempDir(), "missing", "tickets.json")
	if _, e := s.Handoff(context.Background(), Config{"https://jira.test", "u", "p", "P", "Task"}, "id", "t", "b", ""); e == nil || !strings.Contains(e.Error(), "no request sent") {
		t.Fatal(e)
	}
}
func TestLostReplyAndRedirect(t *testing.T) {
	for _, redirect := range []bool{false, true} {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if redirect {
				w.Header().Set("Location", "https://should-not-follow.invalid/")
				w.WriteHeader(302)
			} else {
				c, _, _ := w.(http.Hijacker).Hijack()
				c.Close()
			}
		}))
		s, _ := Open(filepath.Join(t.TempDir(), "tickets.json"))
		s.Client = srv.Client()
		s.Client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		_, e := s.Handoff(context.Background(), Config{srv.URL, "u", "p", "P", "Task"}, "id", "t", "b", "")
		if e == nil || s.Records[srv.URL+"|id"].State != "uncertain" {
			t.Fatal(e)
		}
		srv.Close()
	}
}

func TestOriginAliasesShareDuplicateProtection(t *testing.T) {
	for _, origin := range []string{"https://JIRA.EXAMPLE.test/", "https://jira.example.test:443"} {
		got, err := Origin(origin)
		if err != nil || got != "https://jira.example.test" {
			t.Fatalf("origin=%s got=%s err=%v", origin, got, err)
		}
	}
	// Existing files may contain origins written before canonicalization.
	s, err := Open(filepath.Join(t.TempDir(), "tickets.json"))
	if err != nil {
		t.Fatal(err)
	}
	s.Records["https://JIRA.EXAMPLE.test:443|finding"] = Record{State: "uncertain"}
	s.Client = &http.Client{Transport: noNetworkTransport{t}}
	_, err = s.Handoff(context.Background(), Config{"https://jira.example.test", "u", "p", "P", "Task"}, "finding", "title", "body", "")
	if err == nil || !strings.Contains(err.Error(), "may already have succeeded") {
		t.Fatal(err)
	}
}

func TestOriginPortsAndConflictingHistory(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"https://[::1]:443/", "https://[::1]"},
		{"https://[::1]:8443/", "https://[::1]:8443"},
		{" https://JIRA.EXAMPLE.test:8443/ ", "https://jira.example.test:8443"},
	} {
		got, err := Origin(tc.input)
		if err != nil || got != tc.want {
			t.Fatal(tc, got, err)
		}
	}
	s, _ := Open(filepath.Join(t.TempDir(), "tickets.json"))
	s.Client = &http.Client{Transport: noNetworkTransport{t}}
	s.Records["https://jira.example.test|finding"] = Record{Key: "SEC-1", State: "linked"}
	s.Records["https://JIRA.EXAMPLE.test:443|finding"] = Record{State: "uncertain"}
	_, err := s.Handoff(context.Background(), Config{"https://jira.example.test", "u", "p", "P", "Task"}, "finding", "t", "b", "")
	if err == nil || !strings.Contains(err.Error(), "multiple delivery records") {
		t.Fatal(err)
	}
}

type noNetworkTransport struct{ t *testing.T }

func (n noNetworkTransport) RoundTrip(*http.Request) (*http.Response, error) {
	n.t.Error("duplicate protection allowed a network attempt")
	return nil, errors.New("network disabled in test")
}
