package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cartero/internal/storage"
	utils "cartero/internal/utils/string"

	"github.com/gorilla/feeds"
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

type jsonFeed struct {
	Version     string         `json:"version"`
	Title       string         `json:"title"`
	HomePageURL string         `json:"home_page_url,omitempty"`
	FeedURL     string         `json:"feed_url,omitempty"`
	Items       []jsonFeedItem `json:"items"`
}

type jsonFeedItem struct {
	ID            string          `json:"id"`
	URL           string          `json:"url,omitempty"`
	Title         string          `json:"title"`
	Summary       string          `json:"summary,omitempty"`
	ContentText   string          `json:"content_text,omitempty"`
	Image         string          `json:"image,omitempty"`
	DatePublished *time.Time      `json:"date_published,omitempty"`
	Authors       []jsonAuthor    `json:"authors,omitempty"`
	Tags          []string        `json:"tags,omitempty"`
	Cartero       jsonCarteroMeta `json:"_cartero"`
}

type jsonAuthor struct {
	Name string `json:"name"`
}

type jsonCarteroMeta struct {
	Source         string    `json:"source"`
	ReadingMinutes int       `json:"reading_minutes,omitempty"`
	AddedAt        time.Time `json:"added_at"`
}

func (h *Handler) JSONFeed(w http.ResponseWriter, r *http.Request) {
	entries, err := h.entryStore.ListPublishedEntries(r.Context(), h.config.Name, h.config.MaxItems)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	feed := jsonFeed{
		Version:     "https://jsonfeed.org/version/1.1",
		Title:       h.config.SiteName,
		HomePageURL: h.config.SiteURL,
		Items:       make([]jsonFeedItem, 0, len(entries)),
	}
	if h.config.SiteURL != "" {
		feed.FeedURL = h.config.SiteURL + "/feed.json"
	}

	for _, e := range entries {
		item := jsonFeedItem{
			ID:          e.ID,
			URL:         e.Link,
			Title:       e.Title,
			Summary:     e.Description,
			ContentText: e.Content,
			Image:       e.ImageURL,
			Cartero: jsonCarteroMeta{
				Source:         utils.Readable(e.Source),
				ReadingMinutes: readingMinutes(e.Content),
				AddedAt:        e.CreatedAt,
			},
		}
		if !e.PublishedAt.IsZero() {
			item.DatePublished = &e.PublishedAt
		}
		if e.Author != "" {
			item.Authors = []jsonAuthor{{Name: e.Author}}
		}
		if e.MatchedKeywords != "" {
			item.Tags = []string{e.MatchedKeywords}
		}
		feed.Items = append(feed.Items, item)
	}

	body, err := json.Marshal(feed)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sum := sha256.Sum256(body)
	etag := `"` + hex.EncodeToString(sum[:16]) + `"`
	w.Header().Set("Content-Type", "application/feed+json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = w.Write(body)
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
