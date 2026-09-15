package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const maxIconBytes int64 = 512 << 10

// Private LAN addresses are intentional targets for a NAS dashboard. Loopback,
// link-local (including cloud metadata), multicast and unspecified addresses are not.
func allowedIconIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsValid() && ip.IsGlobalUnicast() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && ip.Zone() == ""
}

func newIconClient() *http.Client {
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	transport := &http.Transport{
		MaxIdleConns: 6, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second,
		TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 3 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			if len(addresses) == 0 {
				return nil, errors.New("host has no addresses")
			}
			for _, ip := range addresses {
				if !allowedIconIP(ip) {
					return nil, errors.New("this address is not allowed for icon fetching")
				}
			}
			for _, ip := range addresses {
				// Dial the validated address directly to avoid a second DNS lookup.
				conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
				if dialErr == nil {
					return conn, nil
				}
				err = dialErr
			}
			return nil, err
		},
	}
	return &http.Client{Transport: transport, Timeout: 4 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 4 {
			return errors.New("too many redirects")
		}
		_, err := parseWebURL(req.URL.String())
		return err
	}}
}

func (a *application) fetchIcon(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL    string `json:"url"`
		Direct bool   `json:"direct"`
	}
	if !readJSON(w, r, &input, 4096) {
		return
	}
	u, err := parseWebURL(input.URL)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	select {
	case a.iconSlots <- struct{}{}:
		defer func() { <-a.iconSlots }()
	default:
		writeError(w, 429, "Icon fetching is busy. Please try again shortly.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	icon, err := discoverIcon(ctx, a.iconClient, u, input.Direct)
	if err != nil {
		writeError(w, 422, "No usable icon found. Upload an image or keep the initials.")
		return
	}
	writeJSON(w, 200, map[string]string{"icon": icon})
}

func fetchBytes(ctx context.Context, client *http.Client, u *url.URL, limit int64) ([]byte, *url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, u, err
	}
	req.Header.Set("User-Agent", "Harbor/1.0 (favicon discovery)")
	req.Header.Set("Accept", "text/html,image/*;q=0.9,*/*;q=0.1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, u, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.Request.URL, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, resp.Request.URL, err
	}
	if int64(len(raw)) > limit {
		return nil, resp.Request.URL, errors.New("response too large")
	}
	return raw, resp.Request.URL, nil
}

func discoverIcon(ctx context.Context, client *http.Client, u *url.URL, direct bool) (string, error) {
	limit := int64(1 << 20)
	if direct {
		limit = maxIconBytes
	}
	raw, finalURL, err := fetchBytes(ctx, client, u, limit)
	if err == nil {
		if icon, imageErr := imageData(raw); imageErr == nil {
			return icon, nil
		}
	}
	if direct {
		return "", errors.New("not a supported image")
	}
	var candidates []*url.URL
	if err == nil {
		candidates = iconLinks(raw, finalURL)
	}
	fallback := *finalURL
	fallback.Path, fallback.RawPath, fallback.RawQuery, fallback.Fragment = "/favicon.ico", "", "", ""
	candidates = append(candidates, &fallback)
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		if seen[candidate.String()] {
			continue
		}
		seen[candidate.String()] = true
		if raw, _, err := fetchBytes(ctx, client, candidate, maxIconBytes); err == nil {
			if icon, err := imageData(raw); err == nil {
				return icon, nil
			}
		}
		if ctx.Err() != nil {
			break
		}
	}
	return "", errors.New("no icon found")
}

// This bounded scanner only extracts link/base attributes; page scripts are
// never executed. Quoted '>' characters and HTML entities are supported.
var tagPattern = regexp.MustCompile(`(?is)<(?:link|base)\b(?:[^>"']|"[^"]*"|'[^']*')*>`)
var attrPattern = regexp.MustCompile(`(?is)([^\s=/>]+)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>` + "`" + `]+))`)
var inactivePattern = regexp.MustCompile(`(?is)<!--.*?-->|<script\b[^>]*>.*?</script\s*>|<style\b[^>]*>.*?</style\s*>`)

func iconLinks(raw []byte, base *url.URL) []*url.URL {
	page := inactivePattern.ReplaceAllString(string(raw), "")
	if end := strings.Index(strings.ToLower(page), "</head>"); end >= 0 {
		page = page[:end]
	}
	tags := tagPattern.FindAllString(page, 64)
	var icons []*url.URL
	baseSeen := false
	for _, tag := range tags {
		attrs := make(map[string]string)
		for _, match := range attrPattern.FindAllStringSubmatch(tag, -1) {
			value := match[2] + match[3] + match[4]
			attrs[strings.ToLower(match[1])] = html.UnescapeString(value)
		}
		ref, err := url.Parse(strings.TrimSpace(attrs["href"]))
		if err != nil || attrs["href"] == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(tag), "<base") && !baseSeen {
			baseSeen = true
			resolved := base.ResolveReference(ref)
			if _, err := parseWebURL(resolved.String()); err == nil {
				base = resolved
			}
			continue
		}
		isIcon := false
		for _, rel := range strings.Fields(strings.ToLower(attrs["rel"])) {
			if rel == "icon" || rel == "apple-touch-icon" {
				isIcon = true
			}
		}
		resolved := base.ResolveReference(ref)
		if _, err := parseWebURL(resolved.String()); isIcon && err == nil {
			icons = append(icons, resolved)
			if len(icons) >= 3 {
				break
			}
		}
	}
	return icons
}

func imageData(raw []byte) (string, error) {
	if len(raw) == 0 || int64(len(raw)) > maxIconBytes {
		return "", errors.New("Use an image smaller than 512 KB.")
	}
	mime := http.DetectContentType(raw)
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/x-icon", "image/vnd.microsoft.icon":
	default:
		if err := safeSVG(raw); err != nil {
			return "", errors.New("Use a PNG, JPEG, GIF, WebP, ICO, or simple SVG image.")
		}
		mime = "image/svg+xml"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

func validateIconData(value string) error {
	if int64(len(value)) > maxIconBytes*2 {
		return errors.New("The icon is too large.")
	}
	header, body, ok := strings.Cut(value, ",")
	if !ok || !strings.HasPrefix(header, "data:image/") || !strings.HasSuffix(header, ";base64") {
		return errors.New("Upload or fetch an image before saving.")
	}
	raw, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return errors.New("The icon data is invalid.")
	}
	canonical, err := imageData(raw)
	if err != nil {
		return err
	}
	actualHeader, _, _ := strings.Cut(canonical, ",")
	if actualHeader != header {
		return errors.New("The icon format does not match its contents.")
	}
	return nil
}

func safeSVG(raw []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	allowed := " svg g path rect circle ellipse line polyline polygon defs lineargradient radialgradient stop clippath mask pattern symbol use title desc "
	depth, roots, elements := 0, 0, 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch t := token.(type) {
		case xml.StartElement:
			name := strings.ToLower(t.Name.Local)
			if depth == 0 {
				roots++
				if name != "svg" || roots != 1 {
					return errors.New("not SVG")
				}
			}
			depth++
			elements++
			if depth > 64 || elements > 10000 || !strings.Contains(allowed, " "+name+" ") {
				return errors.New("unsupported SVG element")
			}
			for _, attr := range t.Attr {
				key, value := strings.ToLower(attr.Name.Local), strings.ToLower(strings.TrimSpace(attr.Value))
				if strings.HasPrefix(key, "on") || strings.ContainsAny(value, "\\\x00") || strings.Contains(value, "@import") || strings.Contains(value, "javascript:") {
					return errors.New("active SVG content")
				}
				if key == "href" && !strings.HasPrefix(value, "#") {
					return errors.New("external SVG reference")
				}
				if strings.Contains(value, "url(") && !(strings.HasPrefix(value, "url(#") && strings.HasSuffix(value, ")") && strings.Count(value, "url(") == 1) {
					return errors.New("external SVG resource")
				}
			}
		case xml.EndElement:
			depth--
		case xml.Directive:
			return errors.New("SVG directives are not allowed")
		case xml.ProcInst:
			if t.Target != "xml" {
				return errors.New("SVG processing instructions are not allowed")
			}
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(t)) != "" {
				return errors.New("unexpected text")
			}
		}
	}
	if roots != 1 || depth != 0 {
		return errors.New("incomplete SVG")
	}
	return nil
}
