package desktop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"
)

func DataDir(name string) (string, error) {
	d := os.Getenv("PORTFOLIO_DATA_DIR")
	if d == "" {
		var e error
		d, e = os.UserConfigDir()
		if e != nil {
			return "", e
		}
	}
	d = filepath.Join(d, name)
	return d, os.MkdirAll(d, 0700)
}
func Token() string {
	b := make([]byte, 24)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func Protect(host, token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
		if r.Host != host || r.URL.Query().Get("token") != token {
			http.Error(w, "Open the private launcher URL", 403)
			return
		}
		if r.Method != "GET" && r.Method != "POST" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		if r.Method == "POST" {
			if o := r.Header.Get("Origin"); o != "" && o != "http://"+host {
				http.Error(w, "Invalid origin", 403)
				return
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				http.Error(w, "Cross-site request blocked", 403)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 128<<20)
		}
		next.ServeHTTP(w, r)
	})
}
func Serve(name, dir string, factory func(string) http.Handler) error {
	lock := filepath.Join(dir, "application.lock")
	f, e := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return fmt.Errorf("another instance may be running; after a crash confirm no instance remains before deleting %s", lock)
	}
	fmt.Fprintln(f, os.Getpid())
	f.Close()
	defer os.Remove(lock)
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		return e
	}
	token := Token()
	address := "http://" + l.Addr().String() + "/?token=" + token
	server := &http.Server{Handler: Protect(l.Addr().String(), token, factory(token)), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	fmt.Println(name + ": " + address)
	fmt.Println("Data: " + dir + "\nClose with Ctrl+C in this console. Closing the browser does not stop the application.")
	if os.Getenv("PORTFOLIO_NO_BROWSER") == "" {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", address)
		case "darwin":
			cmd = exec.Command("open", address)
		default:
			cmd = exec.Command("xdg-open", address)
		}
		if e = cmd.Start(); e == nil {
			go cmd.Wait()
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() { <-ctx.Done(); server.Close() }()
	e = server.Serve(l)
	if errors.Is(e, http.ErrServerClosed) {
		return nil
	}
	return e
}

const Style = `body{font:16px system-ui;background:#0b1627;color:#e4edf8;max-width:1100px;margin:40px auto;padding:0 24px}section{background:#18293e;padding:24px;border:1px solid #344a64;border-radius:12px;margin:20px 0}h1{font-size:38px}h2{font-size:23px}p,small{line-height:1.6}small{color:#b8cce0}input,textarea,select{display:block;width:94%;max-width:800px;background:#0b1627;color:white;border:1px solid #8ba3bd;padding:10px;border-radius:5px;margin:8px 0}textarea{min-height:120px}label{display:block;margin:16px 0}button{background:#92e5c0;color:#102332;font-weight:700;padding:12px 18px;border:0;border-radius:6px;cursor:pointer;margin:8px 4px 8px 0}a{color:#a3d2ff}pre{white-space:pre-wrap;overflow-wrap:anywhere}td,th{padding:10px;text-align:left;border-bottom:1px solid #42566d;overflow-wrap:anywhere}table{border-collapse:collapse;width:100%}.notice{padding:18px;background:#29415b;border-radius:8px}.scroll{overflow:auto}summary{cursor:pointer}`
