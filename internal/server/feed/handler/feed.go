package handler

import (
	"fmt"
	"net/http"
	"time"

	"cartero/internal/storage"

	"github.com/gorilla/feeds"
)

func (h *Handler) ServiceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Service-Worker-Allowed", "/")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, "assets/sw.js")
}

func (h *Handler) RSSFeed(w http.ResponseWriter, r *http.Request) {
	entries, err := h.entryStore.ListPublishedEntries(r.Context(), h.config.Name, h.config.FeedSize)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "Error: %v", err)
		return
	}

	feed := h.buildFeed(entries)
	rss, err := feed.ToRss()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = fmt.Fprint(w, rss)
}

func (h *Handler) AtomFeed(w http.ResponseWriter, r *http.Request) {
	entries, err := h.entryStore.ListPublishedEntries(r.Context(), h.config.Name, h.config.FeedSize)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "Error: %v", err)
		return
	}

	feed := h.buildFeed(entries)
	atom, err := feed.ToAtom()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = fmt.Fprint(w, atom)
}

func (h *Handler) JSONFeed(w http.ResponseWriter, r *http.Request) {
	entries, err := h.entryStore.ListPublishedEntries(r.Context(), h.config.Name, h.config.FeedSize)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "Error: %v", err)
		return
	}

	feed := h.buildFeed(entries)
	jsonStr, err := feed.ToJSON()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = fmt.Fprint(w, jsonStr)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"status":"ok","name":"%s","time":"%s"}`, h.config.Name, time.Now().UTC().Format(time.RFC3339))
}

func (h *Handler) buildFeed(entries []storage.FeedEntry) *feeds.Feed {
	items := make([]*feeds.Item, 0, len(entries))

	for _, entry := range entries {
		item := &feeds.Item{
			Id:          entry.ID,
			Title:       entry.Title,
			Link:        &feeds.Link{Href: entry.Link},
			Description: entry.Description,
			Content:     entry.Content,
			Author:      &feeds.Author{Name: entry.Author},
			Created:     entry.PublishedAt,
		}
		items = append(items, item)
	}

	if len(items) > h.config.MaxItems {
		items = items[:h.config.MaxItems]
	}

	return &feeds.Feed{
		Title:       fmt.Sprintf("Cartero Feed (%s)", h.config.Name),
		Link:        &feeds.Link{Href: "http://localhost/"},
		Description: "Content aggregation feed from Cartero",
		Author:      &feeds.Author{Name: "Cartero"},
		Created:     time.Now().UTC(),
		Items:       items,
	}
}
