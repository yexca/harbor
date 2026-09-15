package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

//go:embed web/*
var assets embed.FS

type application struct {
	store      *store
	password   string
	sessions   map[string]time.Time
	sessionMu  sync.Mutex
	iconSlots  chan struct{}
	loginSlots chan struct{}
	iconClient *http.Client
}

func main() {
	addr := env("HARBOR_ADDR", ":8080")
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		if strings.HasPrefix(addr, ":") {
			addr = "127.0.0.1" + addr
		}
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://" + addr + "/healthz")
		if err != nil {
			os.Exit(1)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		return
	}
	s, err := openStore(env("HARBOR_DATA_DIR", "./data"))
	if err != nil {
		log.Fatal(err)
	}
	a := newApplication(s, os.Getenv("HARBOR_ADMIN_PASSWORD"))
	server := &http.Server{Addr: addr, Handler: a.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdown)
	}()
	log.Printf("Harbor is listening on %s; data: %s; edit protection: %t", addr, s.path, a.password != "")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func newApplication(s *store, password string) *application {
	return &application{store: s, password: password, sessions: make(map[string]time.Time), iconSlots: make(chan struct{}, 3), loginSlots: make(chan struct{}, 3), iconClient: newIconClient()}
}

func (a *application) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok\n")) })
	mux.HandleFunc("GET /api/session", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]bool{"protected": a.password != "", "canEdit": a.authorized(r)})
	})
	mux.HandleFunc("POST /api/login", a.login)
	mux.HandleFunc("POST /api/logout", a.logout)
	mux.HandleFunc("GET /api/services", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, a.store.list()) })
	mux.Handle("POST /api/services", a.editor(http.HandlerFunc(a.createService)))
	mux.Handle("PUT /api/services/{id}", a.editor(http.HandlerFunc(a.updateService)))
	mux.Handle("PATCH /api/services/{id}/visibility", a.editor(http.HandlerFunc(a.updateVisibility)))
	mux.Handle("DELETE /api/services/{id}", a.editor(http.HandlerFunc(a.deleteService)))
	mux.Handle("POST /api/icon", a.editor(http.HandlerFunc(a.fetchIcon)))
	web, _ := fs.Sub(assets, "web")
	files := http.FileServer(http.FS(web))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/app.js" && r.URL.Path != "/style.css" && r.URL.Path != "/favicon.svg" && r.URL.Path != "/wallpaper.jpg" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; font-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Host != r.Host || (u.Scheme != "http" && u.Scheme != "https") {
					writeError(w, 403, "Cross-origin changes are not allowed.")
					return
				}
			}
			if r.Header.Get("X-Harbor-Request") != "1" || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				writeError(w, 403, "Missing same-origin request header.")
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}

func (a *application) authorized(r *http.Request) bool {
	if a.password == "" {
		return true
	}
	cookie, err := r.Cookie("harbor_session")
	if err != nil {
		return false
	}
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()
	now := time.Now()
	for token, expiry := range a.sessions {
		if !now.Before(expiry) {
			delete(a.sessions, token)
		}
	}
	return now.Before(a.sessions[cookie.Value])
}

func (a *application) editor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.authorized(r) {
			writeError(w, 401, "Unlock editing to make changes.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *application) login(w http.ResponseWriter, r *http.Request) {
	select {
	case a.loginSlots <- struct{}{}:
		defer func() { <-a.loginSlots }()
	default:
		writeError(w, 429, "Too many sign-in attempts. Please try again shortly.")
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	if !readJSON(w, r, &input, 4096) {
		return
	}
	got, want := sha256.Sum256([]byte(input.Password)), sha256.Sum256([]byte(a.password))
	if a.password != "" && subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
		time.Sleep(time.Second)
		writeError(w, 401, "That password is incorrect.")
		return
	}
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		writeError(w, 500, "Could not start a session.")
		return
	}
	key := hex.EncodeToString(token)
	a.sessionMu.Lock()
	if len(a.sessions) >= 1000 {
		clear(a.sessions)
	}
	a.sessions[key] = time.Now().Add(24 * time.Hour)
	a.sessionMu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "harbor_session", Value: key, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: requestIsHTTPS(r), MaxAge: 86400})
	writeJSON(w, 200, map[string]bool{"canEdit": true})
}

func requestIsHTTPS(r *http.Request) bool {
	// The browser's Origin also works behind a TLS-terminating reverse proxy.
	return r.TLS != nil || strings.HasPrefix(r.Header.Get("Origin"), "https://")
}

func (a *application) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("harbor_session"); err == nil {
		a.sessionMu.Lock()
		delete(a.sessions, cookie.Value)
		a.sessionMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "harbor_session", Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: requestIsHTTPS(r), MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func (a *application) createService(w http.ResponseWriter, r *http.Request) {
	var item service
	if !readJSON(w, r, &item, maxIconBytes*2) {
		return
	}
	if err := validateService(&item); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	item, err := a.store.add(item)
	if err != nil {
		log.Printf("save service: %v", err)
		writeError(w, 500, "Could not save the service. Check the data directory and service limit.")
		return
	}
	writeJSON(w, 201, item)
}

func (a *application) updateService(w http.ResponseWriter, r *http.Request) {
	var item service
	if !readJSON(w, r, &item, maxIconBytes*2) {
		return
	}
	if err := validateService(&item); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	item.ID = r.PathValue("id")
	if err := a.store.update(item.ID, item); err != nil {
		a.storeError(w, err)
		return
	}
	writeJSON(w, 200, item)
}

func (a *application) deleteService(w http.ResponseWriter, r *http.Request) {
	if err := a.store.remove(r.PathValue("id")); err != nil {
		a.storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *application) updateVisibility(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Hidden *bool `json:"hidden"`
	}
	if !readJSON(w, r, &input, 1024) {
		return
	}
	if input.Hidden == nil {
		writeError(w, 400, "Specify whether the service is hidden from home.")
		return
	}
	item, err := a.store.setVisibility(r.PathValue("id"), *input.Hidden)
	if err != nil {
		a.storeError(w, err)
		return
	}
	writeJSON(w, 200, item)
}

func (a *application) storeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errNotFound) {
		writeError(w, 404, "This service no longer exists. Refresh and try again.")
		return
	}
	log.Printf("save configuration: %v", err)
	writeError(w, 500, "Could not save your changes. Check the data directory permissions and free space.")
}

func validateService(item *service) error {
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	if len([]rune(item.Name)) < 1 || len([]rune(item.Name)) > 80 {
		return errors.New("Enter a service name of 1 to 80 characters.")
	}
	if len([]rune(item.Description)) > 160 {
		return errors.New("Keep the description under 160 characters.")
	}
	u, err := parseWebURL(item.URL)
	if err != nil {
		return err
	}
	item.URL = u.String()
	if item.Icon != "" {
		if err := validateIconData(item.Icon); err != nil {
			return err
		}
	}
	return nil
}

func parseWebURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) > 2048 {
		return nil, errors.New("The address is too long.")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || strings.ContainsAny(u.Host, " \t\r\n\\") {
		return nil, errors.New("Enter a valid http:// or https:// address without embedded credentials.")
	}
	return u, nil
}

func readJSON(w http.ResponseWriter, r *http.Request, target any, limit int64) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, 415, "Send a JSON request.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, 400, "Invalid or oversized request.")
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(w, 400, "Send exactly one JSON object.")
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}
