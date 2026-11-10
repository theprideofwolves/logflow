package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/theprideofwolves/logflow/internal/ticket"
)

func postFields(h http.Handler, fields map[string]string) *httptest.ResponseRecorder {
	var body bytes.Buffer
	m := multipart.NewWriter(&body)
	for k, v := range fields {
		m.WriteField(k, v)
	}
	m.Close()
	r := httptest.NewRequest("POST", "http://localhost/?token=t", &body)
	r.Header.Set("Content-Type", m.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestInvalidActionsPreserveLatestReport(t *testing.T) {
	dir := t.TempDir()
	store, err := ticket.Open(filepath.Join(dir, "tickets.json"))
	if err != nil {
		t.Fatal(err)
	}
	h := newUI(dir, "t", store, saved{})
	if w := postFields(h, map[string]string{"action": "demo"}); w.Code != 200 {
		t.Fatal(w.Code)
	}
	before, err := os.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fields := range []map[string]string{
		{"action": "analyze", "threshold": "not an integer", "window": "10m"},
		{"action": "analyze", "threshold": "6", "window": "bad"},
		{"action": "analyze", "threshold": "6", "window": "10m"},
		{"action": "unknown"},
		{"action": "ticket", "id": "missing", "jira": "https://never-called.invalid"},
	} {
		postFields(h, fields)
		after, err := os.ReadFile(filepath.Join(dir, "report.json"))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("invalid action replaced report", err)
		}
	}
	// A persistence failure must also preserve the in-memory export.
	if err := os.Rename(dir, dir+"-moved"); err != nil {
		t.Fatal(err)
	}
	defer os.Rename(dir+"-moved", dir)
	w := postFields(h, map[string]string{"action": "demo"})
	if !strings.Contains(w.Body.String(), "previous report retained") {
		t.Fatal(w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/export?token=t", nil))
	var want, got saved
	if json.Unmarshal(before, &want) != nil || json.Unmarshal(w.Body.Bytes(), &got) != nil || want.Report.InputHash != got.Report.InputHash || want.Report.At != got.Report.At {
		t.Fatal("save failure replaced current result")
	}
}

func TestThemesAcrossPagesAndProtectedPreference(t *testing.T) {
	dir := t.TempDir()
	store, _ := ticket.Open(filepath.Join(dir, "tickets.json"))
	h := newUI(dir, "t", store, saved{})
	for _, theme := range []string{"light", "dark"} {
		for _, path := range []string{"/", "/welcome", "/setup"} {
			r := httptest.NewRequest("GET", "http://localhost"+path+"?token=t", nil)
			r.AddCookie(&http.Cookie{Name: "logflow_theme", Value: theme})
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 200 || !strings.Contains(w.Body.String(), `data-theme="`+theme+`"`) || strings.Contains(w.Body.String(), "ZgotmplZ") || !strings.Contains(w.Body.String(), `action="/theme?token=t"`) {
				t.Fatal(path, theme, w.Code, w.Body.String())
			}
		}
	}
	for _, origin := range []string{"https://evil.test", "null"} {
		r := httptest.NewRequest("POST", "http://localhost/theme?token=t", strings.NewReader("theme=dark"))
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 || len(w.Result().Cookies()) != 0 {
			t.Fatal(w.Code, w.Header())
		}
	}
	r := httptest.NewRequest("POST", "http://localhost/theme", strings.NewReader("theme=dark"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
	postFields(h, map[string]string{"action": "demo"})
	r = httptest.NewRequest("GET", "http://localhost/?token=t", nil)
	r.AddCookie(&http.Cookie{Name: "logflow_theme", Value: "dark"})
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), `data-theme="dark"`) || strings.Contains(w.Body.String(), `name="secret"`) || !strings.Contains(w.Body.String(), "Demo alerts cannot create Jira tickets") {
		t.Fatal("workspace theme or demo isolation", w.Body.String())
	}
}

func TestConcurrentExportsAndDemo(t *testing.T) {
	dir := t.TempDir()
	store, _ := ticket.Open(filepath.Join(dir, "tickets.json"))
	h := newUI(dir, "t", store, saved{})
	postFields(h, map[string]string{"action": "demo"})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				postFields(h, map[string]string{"action": "demo"})
				return
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/export?token=t", nil))
			var result saved
			if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil || result.Report == nil || len(result.Report.Alerts) != 1 {
				t.Error("inconsistent concurrent export")
			}
		}(i)
	}
	wg.Wait()
}

func TestSavedTicketTextEscaped(t *testing.T) {
	dir := t.TempDir()
	store, _ := ticket.Open(filepath.Join(dir, "tickets.json"))
	store.Records["local"] = ticket.Record{Key: "<script>alert(1)</script>", State: "linked"}
	h := newUI(dir, "t", store, saved{})
	postFields(h, map[string]string{"action": "demo"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/?token=t", nil))
	if strings.Contains(w.Body.String(), "<script>") || !strings.Contains(w.Body.String(), "&lt;script&gt;") {
		t.Fatal("stored text was not escaped")
	}
}
