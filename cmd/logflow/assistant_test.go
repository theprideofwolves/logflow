package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/theprideofwolves/logflow/internal/analysis"
	"github.com/theprideofwolves/logflow/internal/ticket"
)

func TestDraftUIAndApprovalBoundaries(t *testing.T) {
	r, err := analysis.Process(context.Background(), strings.NewReader(analysis.Demo()), analysis.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		w.WriteHeader(201)
		fmt.Fprint(w, `{"key":"SEC-1"}`)
	}))
	defer srv.Close()
	dir := t.TempDir()
	store, _ := ticket.Open(filepath.Join(dir, "tickets.json"))
	store.Client = srv.Client()
	h := newUI(dir, "t", store, saved{Report: r})
	w := postFields(h, map[string]string{"action": "draft", "provider": "offline", "id": r.Alerts[0].ID})
	if !strings.Contains(w.Body.String(), "Investigation draft") || !strings.Contains(w.Body.String(), "auth-scope") {
		t.Fatal(w.Body.String())
	}
	w = postFields(h, map[string]string{"action": "draft", "provider": "openai", "id": r.Alerts[0].ID})
	if !strings.Contains(w.Body.String(), "Confirm the external-provider disclosure") {
		t.Fatal("remote opt-in bypass")
	}
	w = postFields(h, map[string]string{"action": "approve_ticket", "approval": "invented", "confirm_send": "yes", "secret": "fixture"})
	if calls.Load() != 0 {
		t.Fatal("approval bypass")
	}
	prepare := func() string {
		t.Helper()
		w := postFields(h, map[string]string{"action": "ticket", "id": r.Alerts[0].ID, "jira": srv.URL, "email": "analyst@example.test", "project": "SEC", "kind": "Task"})
		m := regexp.MustCompile(`name="approval" value="([a-f0-9]+)"`).FindStringSubmatch(w.Body.String())
		if len(m) != 2 {
			t.Fatal(w.Body.String())
		}
		return m[1]
	}
	nonce := prepare()
	if calls.Load() != 0 {
		t.Fatal("preview performed a write")
	}
	postFields(h, map[string]string{"action": "approve_ticket", "approval": nonce, "secret": "fixture"})
	if calls.Load() != 0 {
		t.Fatal("missing approval accepted")
	}
	postFields(h, map[string]string{"action": "approve_ticket", "approval": nonce, "confirm_send": "yes", "secret": "fixture"})
	if calls.Load() != 1 {
		t.Fatal("approved create missing")
	}
	postFields(h, map[string]string{"action": "approve_ticket", "approval": nonce, "confirm_send": "yes", "secret": "fixture"})
	if calls.Load() != 1 {
		t.Fatal("approval reused")
	}
	nonce = prepare()
	r.InputHash = "changed snapshot"
	postFields(h, map[string]string{"action": "approve_ticket", "approval": nonce, "confirm_send": "yes", "secret": "fixture"})
	if calls.Load() != 1 {
		t.Fatal("stale snapshot accepted")
	}
}
