package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/theprideofwolves/logflow/internal/analysis"
	"github.com/theprideofwolves/logflow/internal/ticket"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplayCLIAndUI(t *testing.T) {
	var out, errs bytes.Buffer
	if code := run([]string{"-replay"}, &out, &errs); code != 0 {
		t.Fatal(code, errs.String())
	}
	var results []analysis.ReplayResult
	if json.Unmarshal(out.Bytes(), &results) != nil || len(results) != 4 {
		t.Fatal(out.String())
	}
	for _, args := range [][]string{{"-replay", "input.jsonl"}, {"-replay", "-threshold", "3"}} {
		if run(args, &bytes.Buffer{}, &bytes.Buffer{}) != 2 {
			t.Fatal(args)
		}
	}
	dir := t.TempDir()
	store, _ := ticket.Open(filepath.Join(dir, "tickets.json"))
	h := newUI(dir, "t", store, saved{})
	w := postFields(h, map[string]string{"action": "demo"})
	if !strings.Contains(w.Body.String(), "How sensitive is this finding?") || !strings.Contains(w.Body.String(), "not the full input") {
		t.Fatal(w.Body.String())
	}
	before := httptest.NewRecorder()
	h.ServeHTTP(before, httptest.NewRequest("GET", "http://localhost/export?token=t", nil))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/replay?token=t", nil))
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &results) != nil || len(results) != 4 {
		t.Fatal(w.Code, w.Body.String())
	}
	after := httptest.NewRecorder()
	h.ServeHTTP(after, httptest.NewRequest("GET", "http://localhost/export?token=t", nil))
	if !bytes.Equal(before.Body.Bytes(), after.Body.Bytes()) {
		t.Fatal("replay replaced investigation")
	}
}

func TestLegacyReportSensitivityHint(t *testing.T) {
	r, err := analysis.Process(context.Background(), strings.NewReader(analysis.Demo()), analysis.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	r.Alerts[0].Sensitivity = nil
	dir := t.TempDir()
	store, _ := ticket.Open(filepath.Join(dir, "tickets.json"))
	h := newUI(dir, "t", store, saved{Report: r, Demo: true})
	// A completed guide is not needed to reach a persisted export; starting own
	// data opens the existing workspace without analyzing or replacing it.
	start := httptest.NewRequest("POST", "http://localhost/onboarding/start?token=t", strings.NewReader("mode=resume"))
	start.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(httptest.NewRecorder(), start)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/?token=t", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "older report has no sensitivity table") {
		t.Fatal(w.Code, w.Body.String())
	}
}
