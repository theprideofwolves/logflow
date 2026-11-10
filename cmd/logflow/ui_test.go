package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theprideofwolves/logflow/internal/analysis"
	"github.com/theprideofwolves/logflow/internal/ticket"
)

func TestIncompleteResultsVisible(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "line limit", true: "cancelled"}[cancelled], func(t *testing.T) {
			dir := t.TempDir()
			store, err := ticket.Open(filepath.Join(dir, "tickets.json"))
			if err != nil {
				t.Fatal(err)
			}
			h := newUI(dir, "token", store, saved{})
			var body bytes.Buffer
			m := multipart.NewWriter(&body)
			m.WriteField("action", "analyze")
			m.WriteField("threshold", "6")
			m.WriteField("window", "10m")
			f, err := m.CreateFormFile("logs", "synthetic.jsonl")
			if err != nil {
				t.Fatal(err)
			}
			if cancelled {
				f.Write([]byte(analysis.Demo()))
			} else {
				f.Write([]byte(strings.Repeat("x", 2<<20)))
			}
			m.Close()
			r := httptest.NewRequest("POST", "http://localhost/?token=token", &body)
			r.Header.Set("Content-Type", m.FormDataContentType())
			if cancelled {
				ctx, cancel := context.WithCancel(r.Context())
				cancel()
				r = r.WithContext(ctx)
			}
			out := httptest.NewRecorder()
			h.ServeHTTP(out, r)
			reason := "line exceeds 1 MiB"
			if cancelled {
				reason = "cancelled; partial evidence"
			}
			if out.Code != 200 || !strings.Contains(out.Body.String(), "Complete: false") || !strings.Contains(out.Body.String(), reason) || !strings.Contains(out.Body.String(), "partial-prefix:") {
				t.Fatal(out.Code, out.Body.String())
			}
			if _, err := os.Stat(filepath.Join(dir, "onboarding.json")); !os.IsNotExist(err) {
				t.Fatal("incomplete analysis marked guide complete", err)
			}
		})
	}
}

func TestOwnLogUploadAndCompletionBoundary(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "partial"}[partial], func(t *testing.T) {
			dir := t.TempDir()
			ts, err := ticket.Open(filepath.Join(dir, "tickets.json"))
			if err != nil {
				t.Fatal(err)
			}
			h := newUI(dir, "token", ts, saved{})
			var body bytes.Buffer
			m := multipart.NewWriter(&body)
			m.WriteField("action", "analyze")
			m.WriteField("threshold", "6")
			m.WriteField("window", "10m")
			file, err := m.CreateFormFile("logs", "ordered.jsonl")
			if err != nil {
				t.Fatal(err)
			}
			logs := analysis.Demo()
			if partial {
				logs += "not-json\n"
			}
			file.Write([]byte(logs))
			m.Close()
			r := httptest.NewRequest("POST", "http://localhost/?token=token", &body)
			r.Header.Set("Content-Type", m.FormDataContentType())
			out := httptest.NewRecorder()
			h.ServeHTTP(out, r)
			if out.Code != 200 {
				t.Fatal(out.Code, out.Body.String())
			}
			out = httptest.NewRecorder()
			h.ServeHTTP(out, httptest.NewRequest("GET", "http://localhost/export?token=token", nil))
			var result saved
			if err := json.Unmarshal(out.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Demo || result.Report == nil || result.Report.Complete == partial || len(result.Report.Alerts) != 1 || len(result.Report.Alerts[0].Evidence) != 6 {
				t.Fatal("incorrect own-log investigation", out.Body.String())
			}
			_, err = os.Stat(filepath.Join(dir, "onboarding.json"))
			if partial && !os.IsNotExist(err) {
				t.Fatal("partial upload marked complete", err)
			}
			if !partial && err != nil {
				t.Fatal("complete upload progress not saved", err)
			}
		})
	}
}

func TestDemoPersistenceExportAndIsolation(t *testing.T) {
	dir := t.TempDir()
	ts, _ := ticket.Open(filepath.Join(dir, "tickets.json"))
	h := newUI(dir, "token", ts, saved{})
	var b bytes.Buffer
	m := multipart.NewWriter(&b)
	m.WriteField("action", "demo")
	m.Close()
	r := httptest.NewRequest("POST", "http://localhost/?token=token", &b)
	r.Header.Set("Content-Type", m.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "SYNTHETIC") || !strings.Contains(w.Body.String(), `data-theme="light"`) {
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
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/export?token=new-token", nil))
	if !strings.Contains(w.Body.String(), state.Report.InputHash) {
		t.Fatal("export lost evidence")
	}
	b.Reset()
	m = multipart.NewWriter(&b)
	m.WriteField("action", "ticket")
	m.WriteField("id", state.Report.Alerts[0].ID)
	m.Close()
	r = httptest.NewRequest("POST", "http://localhost/?token=new-token", &b)
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
