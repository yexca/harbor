package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func testApp(t *testing.T, password string) *application {
	t.Helper()
	s, err := openStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return newApplication(s, password)
}

func request(t *testing.T, handler http.Handler, method, path string, value any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var body []byte
	if value != nil {
		var err error
		body, err = json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Harbor-Request", "1")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func TestCRUDSurvivesRestart(t *testing.T) {
	a := testApp(t, "")
	handler := a.handler()
	item := service{Name: " Emby ", URL: "http://192.0.2.10:8096", Description: "Movies & TV"}
	w := request(t, handler, "POST", "/api/services", item, nil)
	if w.Code != 201 {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	var created service
	json.Unmarshal(w.Body.Bytes(), &created)
	if created.Name != "Emby" || created.ID == "" {
		t.Fatalf("unexpected item: %+v", created)
	}
	created.Name = "Cinema"
	w = request(t, handler, "PUT", "/api/services/"+created.ID, created, nil)
	if w.Code != 200 {
		t.Fatalf("update: %d %s", w.Code, w.Body)
	}
	reopened, err := openStore(filepath.Dir(a.store.path))
	if err != nil {
		t.Fatal(err)
	}
	if items := reopened.list(); len(items) != 1 || items[0].Name != "Cinema" {
		t.Fatalf("not persisted: %+v", items)
	}
	w = request(t, handler, "DELETE", "/api/services/"+created.ID, nil, nil)
	if w.Code != 204 {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}
	reopened, err = openStore(filepath.Dir(a.store.path))
	if err != nil || len(reopened.list()) != 0 {
		t.Fatal("deletion was not persisted", err)
	}
	w = request(t, handler, "DELETE", "/api/services/"+created.ID, nil, nil)
	if w.Code != 404 {
		t.Fatalf("missing service: %d", w.Code)
	}
}

func TestConcurrentWritesAndFailedSave(t *testing.T) {
	a := testApp(t, "")
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := a.store.add(service{Name: fmt.Sprintf("Service %d", i), URL: "https://example.com"})
			if err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	reopened, err := openStore(filepath.Dir(a.store.path))
	if err != nil {
		t.Fatal(err)
	}
	if len(reopened.list()) != 20 {
		t.Fatal("a concurrent update was lost")
	}
	before := a.store.list()
	a.store.path = filepath.Join(t.TempDir(), "missing-directory", "services.json")
	if err := a.store.remove(before[0].ID); err == nil {
		t.Fatal("expected a failed write")
	}
	if len(a.store.list()) != 20 {
		t.Fatal("failed write changed memory")
	}
}

func TestHomeVisibilityPreservesExistingServices(t *testing.T) {
	dir := t.TempDir()
	legacy := `{"version":1,"services":[{"id":"legacy-1","name":"Emby","url":"http://nas.example:8096","description":"Movies","icon":""}]}`
	if err := os.WriteFile(filepath.Join(dir, "services.json"), []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.list()[0].Hidden {
		t.Fatal("legacy services should remain on home")
	}
	handler := newApplication(s, "").handler()
	w := request(t, handler, "PATCH", "/api/services/legacy-1/visibility", map[string]bool{"hidden": true}, nil)
	if w.Code != 200 {
		t.Fatalf("hide: %d %s", w.Code, w.Body)
	}
	reopened, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	items := reopened.list()
	if len(items) != 1 || !items[0].Hidden || items[0].Name != "Emby" || items[0].Description != "Movies" {
		t.Fatalf("visibility change lost service data: %+v", items)
	}
	w = request(t, handler, "PATCH", "/api/services/legacy-1/visibility", map[string]bool{"hidden": false}, nil)
	if w.Code != 200 || s.list()[0].Hidden {
		t.Fatal("show on home failed")
	}
	if w = request(t, handler, "PATCH", "/api/services/legacy-1/visibility", map[string]bool{}, nil); w.Code != 400 {
		t.Fatal("missing visibility value should be rejected")
	}
	if w = request(t, handler, "PATCH", "/api/services/missing/visibility", map[string]bool{"hidden": true}, nil); w.Code != 404 {
		t.Fatal("unknown service should be rejected")
	}
	protected := newApplication(s, "secret").handler()
	if w = request(t, protected, "PATCH", "/api/services/legacy-1/visibility", map[string]bool{"hidden": true}, nil); w.Code != 401 {
		t.Fatal("visibility changes require editing permission")
	}
}

func TestCorruptConfigurationPreserved(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "services.json")
	raw := []byte("{incomplete")
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := openStore(dir); err == nil {
		t.Fatal("corrupt config was silently accepted")
	}
	after, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(raw, after) {
		t.Fatal("corrupt config was overwritten")
	}
}

func TestAuthenticationAndRequestProtection(t *testing.T) {
	a := testApp(t, "secret")
	handler := a.handler()
	item := service{Name: "Emby", URL: "http://nas.example:8096"}
	if w := request(t, handler, "POST", "/api/services", item, nil); w.Code != 401 {
		t.Fatalf("unprotected edit: %d", w.Code)
	}
	if w := request(t, handler, "POST", "/api/icon", map[string]string{"url": "http://nas.example:8096"}, nil); w.Code != 401 {
		t.Fatalf("unprotected icon proxy: %d", w.Code)
	}
	if w := request(t, handler, "GET", "/api/services", nil, nil); w.Code != 200 {
		t.Fatal("public viewing failed")
	}
	if w := request(t, handler, "POST", "/api/login", map[string]string{"password": "wrong"}, nil); w.Code != 401 {
		t.Fatal("wrong password accepted")
	}
	w := request(t, handler, "POST", "/api/login", map[string]string{"password": "secret"}, nil)
	if w.Code != 200 || len(w.Result().Cookies()) != 1 {
		t.Fatal("sign-in failed", w.Body)
	}
	cookie := w.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("missing cookie protection")
	}
	if w := request(t, handler, "POST", "/api/services", item, cookie); w.Code != 201 {
		t.Fatal("authorized write failed", w.Body)
	}
	for _, origin := range []string{"https://other.example", "null"} {
		r := httptest.NewRequest("POST", "/api/services", strings.NewReader(`{}`))
		r.Header.Set("Origin", origin)
		r.Header.Set("X-Harbor-Request", "1")
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatalf("cross-origin request accepted: %s", origin)
		}
	}
	r := httptest.NewRequest("POST", "/api/logout", nil)
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("missing request header accepted")
	}
	request(t, handler, "POST", "/api/logout", nil, cookie)
	if w := request(t, handler, "POST", "/api/services", item, cookie); w.Code != 401 {
		t.Fatal("logged-out session still works")
	}
}

func TestInputAndAssetHandling(t *testing.T) {
	handler := testApp(t, "").handler()
	for _, badURL := range []string{"javascript:alert(1)", "file:///etc/passwd", "https://u:p@example.com", "https://", "http://example.com:invalid"} {
		w := request(t, handler, "POST", "/api/services", service{Name: "Bad", URL: badURL}, nil)
		if w.Code != 400 {
			t.Errorf("accepted %q: %d", badURL, w.Code)
		}
	}
	for _, path := range []string{"/", "/app.js", "/style.css", "/favicon.svg", "/wallpaper.jpg", "/healthz"} {
		w := request(t, handler, "GET", path, nil, nil)
		if w.Code != 200 {
			t.Errorf("asset %s: %d", path, w.Code)
		}
		if w.Header().Get("Content-Security-Policy") == "" {
			t.Fatal("missing CSP")
		}
	}
	for _, path := range []string{"/data/services.json", "/main.go", "/missing"} {
		if w := request(t, handler, "GET", path, nil, nil); w.Code != 404 {
			t.Errorf("unexpected file exposed: %s", path)
		}
	}
	for _, raw := range []string{`{"name":"X","url":"https://example.com","unknown":1}`, `{"name":"X","url":"https://example.com"} {}`, strings.Repeat("x", int(maxIconBytes*2+1))} {
		r := httptest.NewRequest("POST", "/api/services", strings.NewReader(raw))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Harbor-Request", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatalf("invalid JSON accepted: %d", w.Code)
		}
	}
}

func TestIconDiscovery(t *testing.T) {
	safe := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="#4565e9" d="M0 0h24v24H0z"/></svg>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app":
			http.Redirect(w, r, "/nested/home", http.StatusFound)
		case "/nested/home":
			io.WriteString(w, `<head><!-- <link rel="icon" href="/wrong"> --><base href="/assets/"><link data-other="a > b" REL="shortcut ICON" href='mark.svg?x=1&amp;y=2'></head>`)
		case "/assets/mark.svg":
			if r.URL.RawQuery != "" && r.URL.Query().Get("y") != "2" {
				t.Error("HTML entities not decoded")
			}
			io.WriteString(w, safe)
		case "/no-icon":
			io.WriteString(w, "<html><head></head></html>")
		case "/favicon.ico":
			io.WriteString(w, safe)
		case "/huge":
			w.Write(bytes.Repeat([]byte("x"), int(maxIconBytes+1)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	for _, path := range []string{"/app", "/no-icon", "/assets/mark.svg"} {
		u, _ := url.Parse(server.URL + path)
		icon, err := discoverIcon(context.Background(), server.Client(), u, path == "/assets/mark.svg")
		if err != nil || !strings.HasPrefix(icon, "data:image/svg+xml;base64,") {
			t.Fatalf("discovery %s: %v", path, err)
		}
		if err := validateIconData(icon); err != nil {
			t.Fatal(err)
		}
	}
	u, _ := url.Parse(server.URL + "/huge")
	if _, err := discoverIcon(context.Background(), server.Client(), u, true); err == nil {
		t.Fatal("oversized image accepted")
	}
}

func TestIconValidation(t *testing.T) {
	for _, raw := range []string{
		`<svg><script>alert(1)</script></svg>`,
		`<svg onload="alert(1)"></svg>`,
		`<svg><use href="https://example.com/x.svg#id"/></svg>`,
		`<!DOCTYPE svg SYSTEM "file:///etc/passwd"><svg/>`,
		`<svg><foreignObject/></svg>`,
		`<svg/><svg/>`,
		`<html><body>Not an icon</body></html>`,
	} {
		if _, err := imageData([]byte(raw)); err == nil {
			t.Errorf("unsafe image accepted: %s", raw)
		}
	}
	for _, addr := range []string{"127.0.0.1", "::1", "::ffff:127.0.0.1", "169.254.169.254", "fe80::1", "0.0.0.0", "224.0.0.1"} {
		if allowedIconIP(netip.MustParseAddr(addr)) {
			t.Errorf("unsafe IP allowed: %s", addr)
		}
	}
	for _, addr := range []string{"192.168.1.10", "10.0.0.3", "172.16.0.2", "fd00::1", "1.1.1.1"} {
		if !allowedIconIP(netip.MustParseAddr(addr)) {
			t.Errorf("expected IP rejected: %s", addr)
		}
	}
}

func pngImage(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 3))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func uploadBackground(t *testing.T, handler http.Handler, raw []byte, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("PUT", "/api/site/background", bytes.NewReader(raw))
	r.Header.Set("Content-Type", "image/png")
	r.Header.Set("X-Harbor-Request", "1")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func TestSiteIdentitySurvivesRestartAndEscapes(t *testing.T) {
	a := testApp(t, "")
	handler := a.handler()
	page := request(t, handler, "GET", "/", nil, nil)
	if page.Code != 200 || !strings.Contains(page.Body.String(), "<title>Harbor</title>") || !strings.Contains(page.Body.String(), `href="/favicon.svg"`) {
		t.Fatalf("default page: %d %s", page.Code, page.Body)
	}
	icon, err := imageData(pngImage(t))
	if err != nil {
		t.Fatal(err)
	}
	w := request(t, handler, "PUT", "/api/site", map[string]string{"title": " Home <NAS> & co ", "icon": icon}, nil)
	var view siteView
	json.Unmarshal(w.Body.Bytes(), &view)
	if w.Code != 200 || view.Title != "Home <NAS> & co" || !view.CustomIcon || !strings.HasPrefix(view.Icon, "/site-icon?v=") {
		t.Fatalf("update site: %d %s", w.Code, w.Body)
	}
	page = request(t, handler, "GET", "/", nil, nil)
	if body := page.Body.String(); !strings.Contains(body, "<title>Home &lt;NAS&gt; &amp; co</title>") || strings.Contains(body, "<NAS>") || !strings.Contains(body, view.Icon) {
		t.Fatalf("title or icon not rendered safely: %s", body)
	}
	served := request(t, handler, "GET", "/site-icon", nil, nil)
	if served.Code != 200 || served.Header().Get("Content-Type") != "image/png" || !strings.Contains(served.Header().Get("Content-Security-Policy"), "sandbox") {
		t.Fatalf("site icon: %d %v", served.Code, served.Header())
	}
	// Omitting the icon keeps it; an invalid update leaves the saved settings unchanged.
	if w := request(t, handler, "PUT", "/api/site", map[string]string{"title": "Den"}, nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"customIcon":true`) {
		t.Fatalf("title-only update: %d %s", w.Code, w.Body)
	}
	if w := request(t, handler, "PUT", "/api/site", map[string]string{"title": strings.Repeat("x", 61)}, nil); w.Code != 400 {
		t.Fatalf("long title accepted: %d", w.Code)
	}
	if w := request(t, handler, "PUT", "/api/site", map[string]string{"title": "Den", "icon": "data:image/png;base64,bm90"}, nil); w.Code != 400 {
		t.Fatalf("invalid icon accepted: %d", w.Code)
	}
	reopened, err := openStore(filepath.Dir(a.store.path))
	if err != nil {
		t.Fatal(err)
	}
	if site := reopened.siteSettings(); site.Title != "Den" || site.Icon != icon {
		t.Fatalf("site not persisted: %+v", site)
	}
	if w := request(t, handler, "PUT", "/api/site", map[string]string{"title": "", "icon": ""}, nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"title":"Harbor"`) || request(t, handler, "GET", "/site-icon", nil, nil).Code != 404 {
		t.Fatalf("reset site: %d %s", w.Code, w.Body)
	}
}

func TestBackgroundReplacementAndRemoval(t *testing.T) {
	a := testApp(t, "")
	handler := a.handler()
	dir := filepath.Dir(a.store.path)
	w := uploadBackground(t, handler, pngImage(t), nil)
	var first siteView
	json.Unmarshal(w.Body.Bytes(), &first)
	if w.Code != 200 || !strings.HasPrefix(first.Background, "/backgrounds/background-") {
		t.Fatalf("upload: %d %s", w.Code, w.Body)
	}
	served := request(t, handler, "GET", first.Background, nil, nil)
	if served.Code != 200 || served.Header().Get("Content-Type") != "image/png" || !bytes.Equal(served.Body.Bytes(), pngImage(t)) {
		t.Fatalf("serve background: %d %v", served.Code, served.Header())
	}
	if page := request(t, handler, "GET", "/", nil, nil); !strings.Contains(page.Body.String(), `data-background="`+first.Background+`"`) {
		t.Fatal("background not rendered into the page")
	}
	// Invalid images and failed saves keep the previous background.
	if w := uploadBackground(t, handler, []byte("<svg/>"), nil); w.Code != 400 {
		t.Fatalf("SVG background accepted: %d", w.Code)
	}
	if w := uploadBackground(t, handler, bytes.Repeat([]byte{0}, int(maxBackgroundBytes)+1), nil); w.Code != 413 {
		t.Fatalf("oversized background accepted: %d", w.Code)
	}
	w = uploadBackground(t, handler, pngImage(t), nil)
	var second siteView
	json.Unmarshal(w.Body.Bytes(), &second)
	if w.Code != 200 || second.Background == first.Background {
		t.Fatalf("replace: %d %s", w.Code, w.Body)
	}
	if request(t, handler, "GET", first.Background, nil, nil).Code != 404 {
		t.Fatal("replaced background is still served")
	}
	if _, err := os.Stat(filepath.Join(dir, strings.TrimPrefix(first.Background, "/backgrounds/"))); !os.IsNotExist(err) {
		t.Fatal("replaced background file was not removed")
	}
	reopened, err := openStore(dir)
	if err != nil || "/backgrounds/"+reopened.siteSettings().Background != second.Background {
		t.Fatalf("background not persisted: %v", err)
	}
	if request(t, handler, "GET", "/backgrounds/..%2Fsite.json", nil, nil).Code != 404 {
		t.Fatal("unexpected file served")
	}
	if w := request(t, handler, "DELETE", "/api/site/background", nil, nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"background":""`) {
		t.Fatalf("remove: %d %s", w.Code, w.Body)
	}
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "background-") || strings.HasPrefix(entry.Name(), ".") {
			t.Fatalf("leftover file: %s", entry.Name())
		}
	}
}

func TestSiteChangesRequireEditing(t *testing.T) {
	a := testApp(t, "secret")
	handler := a.handler()
	if w := request(t, handler, "PUT", "/api/site", map[string]string{"title": "Den"}, nil); w.Code != 401 {
		t.Fatalf("unprotected site update: %d", w.Code)
	}
	if w := uploadBackground(t, handler, pngImage(t), nil); w.Code != 401 {
		t.Fatalf("unprotected background upload: %d", w.Code)
	}
	if w := request(t, handler, "DELETE", "/api/site/background", nil, nil); w.Code != 401 {
		t.Fatalf("unprotected background removal: %d", w.Code)
	}
	if w := request(t, handler, "GET", "/api/site", nil, nil); w.Code != 200 {
		t.Fatalf("public site view: %d", w.Code)
	}
}

func TestCorruptSiteSettingsPreserved(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "site.json")
	raw := []byte(`{"version":1,"title":"Den","background":"../services.json"}`)
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := openStore(dir); err == nil {
		t.Fatal("unsafe background name accepted")
	}
	if after, err := os.ReadFile(file); err != nil || !bytes.Equal(raw, after) {
		t.Fatal("site settings were overwritten")
	}
}
