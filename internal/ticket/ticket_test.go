package ticket

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandoffFailureMatrix(t *testing.T) {
	for _, tc := range []struct {
		name        string
		status      int
		body, state string
	}{{"created", 201, `{"key":"SEC-1"}`, "linked"}, {"invalid reply", 201, `{}`, "uncertain"}, {"permission", 403, `{"secret":"must not echo"}`, "rejected"}, {"rate limit", 429, `{}`, "uncertain"}, {"server", 503, `{}`, "uncertain"}} {
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
		})
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
