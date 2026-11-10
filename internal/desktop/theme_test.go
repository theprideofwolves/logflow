package desktop

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestThemePreference(t *testing.T) {
	for _, tc := range []struct{ value, want string }{{"", "light"}, {"dark", "dark"}, {"light", "light"}, {"invalid", "light"}} {
		r := httptest.NewRequest("GET", "/", nil)
		if tc.value != "" {
			r.AddCookie(&http.Cookie{Name: "logflow_theme", Value: tc.value})
		}
		if got := Theme(r); got != tc.want {
			t.Fatalf("%q: %s", tc.value, got)
		}
	}
	for _, tc := range []struct {
		name, method, theme, target string
		code                        int
		location                    string
	}{
		{"dark", "POST", "dark", "/setup", 303, "/setup?token=private"},
		{"light", "POST", "light", "/welcome", 303, "/welcome?token=private"},
		{"external return", "POST", "dark", "https://evil.test", 303, "/?token=private"},
		{"protocol relative return", "POST", "dark", "//evil.test", 303, "/?token=private"},
		{"get", "GET", "dark", "/", 405, ""},
		{"invalid theme", "POST", "other", "/", 400, ""},
		{"oversized", "POST", strings.Repeat("x", 2048), "/", 400, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := url.Values{"theme": {tc.theme}, "return": {tc.target}}.Encode()
			r := httptest.NewRequest(tc.method, "/theme?token=private", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			if !HandleTheme(w, r) || w.Code != tc.code || w.Header().Get("Location") != tc.location {
				t.Fatal(w.Code, w.Header())
			}
			cookies := w.Result().Cookies()
			if tc.code == 303 {
				if len(cookies) != 1 || cookies[0].Value != tc.theme || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
					t.Fatal(cookies)
				}
			} else if len(cookies) != 0 {
				t.Fatal("invalid request set preference")
			}
		})
	}
}
