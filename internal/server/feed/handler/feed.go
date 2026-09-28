package handler

import (
	"fmt"
	"net/http"
	"time"

	"cartero/internal/storage"
)

func (h *Handler) ServiceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Service-Worker-Allowed", "/")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, "assets/sw.js")
}

func (h *Handler) Robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.ServeFile(w, r, "assets/robots.txt")
}

func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	http.ServeFile(w, r, "assets/sitemap.xml")
}

func (h *Handler) RSSFeed(w http.ResponseWriter, r *http.Request) {
	h.serveFeed(w, r, "feed:rss", min(h.config.FeedSize, h.config.MaxItems), "text/xml; charset=utf-8", h.encodeRSS)
}

func (h *Handler) AtomFeed(w http.ResponseWriter, r *http.Request) {
	h.serveFeed(w, r, "feed:atom", min(h.config.FeedSize, h.config.MaxItems), "text/xml; charset=utf-8", h.encodeAtom)
}

func (h *Handler) JSONFeed(w http.ResponseWriter, r *http.Request) {
	h.serveFeed(w, r, "feed:json", h.config.MaxItems, "application/feed+json; charset=utf-8", h.encodeJSON)
}

func (h *Handler) serveFeed(w http.ResponseWriter, r *http.Request, key string, limit int, contentType string, encode func([]storage.FeedEntry) ([]byte, error)) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=300")

	if e, ok := h.cache.get(key); ok {
		writeHTML(w, r, e)
		return
	}

	entries, err := h.entryStore.ListPublishedEntries(r.Context(), h.config.Name, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	body, err := encode(entries)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeHTML(w, r, h.cache.set(key, body))
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"status":"ok","name":"%s","time":"%s"}`, h.config.Name, time.Now().UTC().Format(time.RFC3339))
}
