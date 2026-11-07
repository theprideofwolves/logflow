package main

import (
	"bytes"
	"encoding/json"
	"github.com/theprideofwolves/logflow/internal/ticket"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDemoPersistenceExportAndIsolation(t *testing.T) {
	dir := t.TempDir()
	ts, _ := ticket.Open(filepath.Join(dir, "tickets.json"))
	h := newUI(dir, "token", ts, saved{})
	var b bytes.Buffer
	m := multipart.NewWriter(&b)
	m.WriteField("action", "demo")
	m.Close()
	r := httptest.NewRequest("POST", "http://localhost/", &b)
	r.Header.Set("Content-Type", m.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "SYNTHETIC") || !strings.Contains(w.Body.String(), "background:#0b1627") {
		t.Fatal(w.Body.String())
	}
	data, e := os.ReadFile(filepath.Join(dir, "report.json"))
	if e != nil {
		t.Fatal(e)
	}
	var state saved
	if json.Unmarshal(data, &state) != nil || len(state.Report.Alerts) != 1 {
		t.Fatal("bad saved report")
	}
	h = newUI(dir, "new-token", ts, state)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/export", nil))
	if !strings.Contains(w.Body.String(), state.Report.InputHash) {
		t.Fatal("export lost evidence")
	}
	b.Reset()
	m = multipart.NewWriter(&b)
	m.WriteField("action", "ticket")
	m.WriteField("id", state.Report.Alerts[0].ID)
	m.Close()
	r = httptest.NewRequest("POST", "http://localhost/", &b)
	r.Header.Set("Content-Type", m.FormDataContentType())
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "synthetic alerts cannot") {
		t.Fatal(w.Body.String())
	}
}
func TestCLIExitSemantics(t *testing.T) {
	var out, errs bytes.Buffer
	if code := run([]string{"-threshold", "0", "missing.jsonl"}, &out, &errs); code != 2 {
		t.Fatal(code)
	}
	if e := saveReport(filepath.Join(t.TempDir(), "missing", "report.json"), saved{}); e == nil {
		t.Fatal("save failure hidden")
	}
}
