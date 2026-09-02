package handler

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

// Proxy fetches the raw HTML of an external page so the browser-side reader
// (Readability.js) can extract it. It exists only because browsers block
// cross-origin fetches (CORS); no extraction happens here.
func (h *Handler) Proxy(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("url")
	if raw == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}

	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u.String(), nil)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CarteroReader/1.0; +https://news.karthihegde.dev)")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "failed to fetch page", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Reject non-HTML responses early (images, PDFs, etc.).
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		if !isHTML(ct) {
			http.Error(w, "not an html page", http.StatusUnsupportedMediaType)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, resp.Body)
}

func isHTML(contentType string) bool {
	for _, p := range []string{"text/html", "application/xhtml+xml", "text/plain", "application/xml"} {
		if len(contentType) >= len(p) && contentType[:len(p)] == p {
			return true
		}
	}
	return false
}
