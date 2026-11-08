package desktop

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogFlowDataDirectoryOverride(t *testing.T) {
	parent := t.TempDir()
	t.Setenv("LOGFLOW_DATA_DIR", parent)
	dir, err := DataDir("LogFlow")
	if err != nil || dir != filepath.Join(parent, "LogFlow") {
		t.Fatal(dir, err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatal("data directory not created", err)
	}
}

func TestRequestBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, method, host, query, origin string
		want                              int
	}{{"allowed", "POST", "127.0.0.1:1234", "?token=private", "http://127.0.0.1:1234", 204}, {"missing token", "GET", "127.0.0.1:1234", "", "", 403}, {"foreign host", "GET", "evil.test", "?token=private", "", 403}, {"foreign origin", "POST", "127.0.0.1:1234", "?token=private", "https://evil.test", 403}, {"method", "DELETE", "127.0.0.1:1234", "?token=private", "", 405}} {
		t.Run(tc.name, func(t *testing.T) {
			h := Protect("127.0.0.1:1234", "private", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
			r := httptest.NewRequest(tc.method, "http://"+tc.host+"/"+tc.query, strings.NewReader(""))
			r.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Header().Get("Referrer-Policy") != "same-origin" {
				t.Fatal("form-compatible referrer policy missing")
			}
			if w.Code != tc.want {
				t.Fatal(w.Code)
			}
		})
	}
}
