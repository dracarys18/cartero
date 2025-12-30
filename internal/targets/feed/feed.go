package feed

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"cartero/internal/cache"
	"cartero/internal/core"
	"cartero/internal/storage"

	"github.com/gorilla/feeds"
)

type Config struct {
	Port     string
	FeedSize int
	MaxItems int
}

type Target struct {
	name      string
	config    Config
	server    *http.Server
	stopCh    chan struct{}
	cache     *cache.Cache[CacheKey, string]
	feedStore storage.FeedStore
}

func NewTarget(name string, config Config, feedStore storage.FeedStore) *Target {
	if config.Port == "" {
		config.Port = "8080"
	}
	if config.FeedSize == 0 {
		config.FeedSize = 100
	}
	if config.MaxItems == 0 {
		config.MaxItems = 50
	}

	return &Target{
		name:      name,
		config:    config,
		stopCh:    make(chan struct{}),
		cache:     NewCache(cache.CacheConfig{TTL: 1 * time.Hour}),
		feedStore: feedStore,
	}
}

func (f *Target) Name() string {
	return f.name
}

func (f *Target) Initialize(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/feed.rss", f.handleRSSFeed)
	mux.HandleFunc("/feed.atom", f.handleAtomFeed)
	mux.HandleFunc("/feed.json", f.handleJSONFeed)
	mux.HandleFunc("/health", f.handleHealth)

	f.server = &http.Server{
		Addr:    ":" + f.config.Port,
		Handler: mux,
	}

	go func() {
		log.Printf("Feed target %s: starting HTTP server on port %s", f.name, f.config.Port)
		if err := f.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Feed target %s: server error: %v", f.name, err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	log.Printf("Feed target %s: listening on http://localhost:%s (endpoints: /feed.rss, /feed.atom, /feed.json, /health)", f.name, f.config.Port)
	return nil
}

func (f *Target) Publish(ctx context.Context, item *core.ProcessedItem) (*core.PublishResult, error) {
	title := "Untitled"
	if t, ok := item.Original.Metadata["title"].(string); ok {
		title = t
	}

	link := ""
	if l, ok := item.Original.Metadata["url"].(string); ok {
		link = l
	} else if l, ok := item.Original.Metadata["link"].(string); ok {
		link = l
	}

	description := ""
	if d, ok := item.Original.Metadata["description"].(string); ok {
		description = d
	}

	author := ""
	if a, ok := item.Original.Metadata["author"].(string); ok {
		author = a
	}

	content := ""
	if s, ok := item.Original.Metadata["summary"].(string); ok {
		content = s
	}

	err := f.feedStore.InsertEntry(ctx, item.Original.ID, title, link, description, content, author, item.Original.Source, item.Original.Timestamp)
	if err != nil {
		log.Printf("Feed target %s: failed to insert entry %s: %v", f.name, item.Original.ID, err)
		return nil, err
	}

	log.Printf("Feed target %s: added entry %s (title=%s)", f.name, item.Original.ID, title)

	f.cache.InvalidatePattern(f.name + ":")

	return &core.PublishResult{
		Success:   true,
		Target:    f.name,
		ItemID:    item.Original.ID,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"feed_type": "rss/atom/json",
		},
	}, nil
}

func (f *Target) Shutdown(ctx context.Context) error {
	if f.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := f.server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Feed target %s: server shutdown error: %v", f.name, err)
		}
	}
	close(f.stopCh)
	return nil
}

func (f *Target) entriesToFeedItems(entries []storage.FeedEntry) []*feeds.Item {
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

	sort.Slice(items, func(i, j int) bool {
		return items[i].Created.After(items[j].Created)
	})

	if len(items) > f.config.MaxItems {
		items = items[:f.config.MaxItems]
	}

	return items
}

func (f *Target) buildFeed(entries []storage.FeedEntry) *feeds.Feed {
	items := f.entriesToFeedItems(entries)
	return &feeds.Feed{
		Title:       fmt.Sprintf("Cartero Feed (%s)", f.name),
		Link:        &feeds.Link{Href: "http://localhost/"},
		Description: "Content aggregation feed from Cartero",
		Author:      &feeds.Author{Name: "Cartero"},
		Created:     time.Now().UTC(),
		Items:       items,
	}
}

func (f *Target) handleRSSFeed(w http.ResponseWriter, r *http.Request) {
	key := NewCacheKey(f.name, TypeRSS)
	if cached, found := f.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		log.Printf("Feed target %s: serving RSS from cache", f.name)
		fmt.Fprint(w, cached)
		return
	}

	entries, err := f.feedStore.ListRecentEntries(r.Context(), f.config.FeedSize)
	if err != nil {
		log.Printf("Feed target %s: failed to list entries: %v", f.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error generating RSS feed: %v", err)
		return
	}

	feed := f.buildFeed(entries)
	rss, err := feed.ToRss()
	if err != nil {
		log.Printf("Feed target %s: failed to generate RSS: %v", f.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error generating RSS feed: %v", err)
		return
	}

	rssKey := NewCacheKey(f.name, TypeRSS)
	f.cache.Set(rssKey, rss)

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	log.Printf("Feed target %s: RSS feed cached (%d bytes)", f.name, len(rss))
	fmt.Fprint(w, rss)
}

func (f *Target) handleAtomFeed(w http.ResponseWriter, r *http.Request) {
	key := NewCacheKey(f.name, TypeAtom)
	if cached, found := f.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		log.Printf("Feed target %s: serving Atom from cache", f.name)
		fmt.Fprint(w, cached)
		return
	}

	entries, err := f.feedStore.ListRecentEntries(r.Context(), f.config.FeedSize)
	if err != nil {
		log.Printf("Feed target %s: failed to list entries: %v", f.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error generating Atom feed: %v", err)
		return
	}

	feed := f.buildFeed(entries)
	atom, err := feed.ToAtom()
	if err != nil {
		log.Printf("Feed target %s: failed to generate Atom: %v", f.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error generating Atom feed: %v", err)
		return
	}

	atomKey := NewCacheKey(f.name, TypeAtom)
	f.cache.Set(atomKey, atom)

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	log.Printf("Feed target %s: Atom feed cached (%d bytes)", f.name, len(atom))
	fmt.Fprint(w, atom)
}

func (f *Target) handleJSONFeed(w http.ResponseWriter, r *http.Request) {
	key := NewCacheKey(f.name, TypeJSON)
	if cached, found := f.cache.Get(key); found {
		w.Header().Set("Content-Type", "application/feed+json; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		log.Printf("Feed target %s: serving JSON from cache", f.name)
		fmt.Fprint(w, cached)
		return
	}

	entries, err := f.feedStore.ListRecentEntries(r.Context(), f.config.FeedSize)
	if err != nil {
		log.Printf("Feed target %s: failed to list entries: %v", f.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error generating JSON feed: %v", err)
		return
	}

	feed := f.buildFeed(entries)
	jsonStr, err := feed.ToJSON()
	if err != nil {
		log.Printf("Feed target %s: failed to generate JSON: %v", f.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error generating JSON feed: %v", err)
		return
	}

	jsonKey := NewCacheKey(f.name, TypeJSON)
	f.cache.Set(jsonKey, jsonStr)

	w.Header().Set("Content-Type", "application/feed+json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	log.Printf("Feed target %s: JSON feed cached (%d bytes)", f.name, len(jsonStr))
	fmt.Fprint(w, jsonStr)
}

func (f *Target) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","name":"%s","time":"%s"}`,
		f.name, time.Now().UTC().Format(time.RFC3339))
}
