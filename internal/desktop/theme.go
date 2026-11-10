package desktop

import (
	"net/http"
	"net/url"
)

// Theme uses a local browser preference; it never changes investigation data.
func Theme(r *http.Request) string {
	if c, err := r.Cookie("logflow_theme"); err == nil && c.Value == "dark" {
		return "dark"
	}
	return "light"
}

// HandleTheme is called only after the launcher's token and origin checks.
func HandleTheme(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/theme" {
		return false
	}
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return true
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	if err := r.ParseForm(); err != nil || (r.PostForm.Get("theme") != "light" && r.PostForm.Get("theme") != "dark") {
		http.Error(w, "Choose light or dark mode", http.StatusBadRequest)
		return true
	}
	path := r.PostForm.Get("return")
	switch path {
	case "/", "/welcome", "/setup":
	default:
		path = "/"
	}
	http.SetCookie(w, &http.Cookie{Name: "logflow_theme", Value: r.PostForm.Get("theme"), Path: "/", MaxAge: 365 * 24 * 60 * 60, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, path+"?token="+url.QueryEscape(r.URL.Query().Get("token")), http.StatusSeeOther)
	return true
}

// ThemeControl is expanded by the page templates. No scripts or external assets.
const ThemeControl = `{{define "theme-control"}}<form class="theme-control" method="post" action="/theme?token={{.Token}}"><input type="hidden" name="return" value="{{.Path}}"><button class="secondary" name="theme" value="{{if eq .Theme "dark"}}light{{else}}dark{{end}}">{{if eq .Theme "dark"}}Light mode{{else}}Dark mode{{end}}</button></form>{{end}}`
