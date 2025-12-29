package feed

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"cartero/internal/cache"
	"cartero/internal/core"

	"github.com/gorilla/feeds"
)

type Config struct {
	Port     string
	FeedSize int
	MaxItems int
}

type Target struct {
	name   string
	config Config
	items  []*feeds.Item
	mu     sync.RWMutex
	server *http.Server
	stopCh chan struct{}
	cache  *cache.Cache[CacheKey, string]
}

func NewTarget(name string, config Config) *Target {
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
		name:   name,
		config: config,
		items:  make([]*feeds.Item, 0, config.FeedSize),
		stopCh: make(chan struct{}),
		cache:  NewCache(cache.CacheConfig{TTL: 1 * time.Hour}),
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
	feedItem := f.convertToFeedItem(item)

	f.mu.Lock()
	f.items = append(f.items, feedItem)
	currentCount := len(f.items)
	if len(f.items) > f.config.FeedSize {
		f.items = f.items[len(f.items)-f.config.FeedSize:]
	}
	f.mu.Unlock()

	log.Printf("Feed target %s: added item %s (title=%s, link=%s). Total items in feed: %d",
		f.name, feedItem.Id, feedItem.Title, feedItem.Link.Href, currentCount)

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

func (f *Target) convertToFeedItem(item *core.ProcessedItem) *feeds.Item {
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

	feedItem := &feeds.Item{
		Id:          item.Original.ID,
		Title:       title,
		Link:        &feeds.Link{Href: link},
		Description: description,
		Content:     content,
		Author:      &feeds.Author{Name: author},
		Created:     item.Original.Timestamp,
	}

	log.Printf("Feed target %s: converted item to feed.Item: id=%s, title=%s, link=%s, pubDate=%s",
		f.name, feedItem.Id, feedItem.Title, link, feedItem.Created.Format(time.RFC3339))

	return feedItem
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

	f.mu.RLock()
	items := make([]*feeds.Item, len(f.items))
	copy(items, f.items)
	totalItems := len(f.items)
	f.mu.RUnlock()

	log.Printf("Feed target %s: handleRSSFeed called. Total items in memory: %d", f.name, totalItems)

	sort.Slice(items, func(i, j int) bool {
		return items[i].Created.After(items[j].Created)
	})

	if len(items) > f.config.MaxItems {
		items = items[:f.config.MaxItems]
	}

	log.Printf("Feed target %s: serving %d items in RSS feed (max_items=%d)", f.name, len(items), f.config.MaxItems)

	feed := &feeds.Feed{
		Title:       fmt.Sprintf("Cartero Feed (%s)", f.name),
		Link:        &feeds.Link{Href: "http://localhost/"},
		Description: "Content aggregation feed from Cartero",
		Author:      &feeds.Author{Name: "Cartero"},
		Created:     time.Now().UTC(),
		Items:       items,
	}

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
	log.Printf("Feed target %s: RSS feed generated and cached (%d bytes)", f.name, len(rss))
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

	f.mu.RLock()
	items := make([]*feeds.Item, len(f.items))
	copy(items, f.items)
	totalItems := len(f.items)
	f.mu.RUnlock()

	log.Printf("Feed target %s: handleAtomFeed called. Total items in memory: %d", f.name, totalItems)

	sort.Slice(items, func(i, j int) bool {
		return items[i].Created.After(items[j].Created)
	})

	if len(items) > f.config.MaxItems {
		items = items[:f.config.MaxItems]
	}

	log.Printf("Feed target %s: serving %d items in Atom feed (max_items=%d)", f.name, len(items), f.config.MaxItems)

	feed := &feeds.Feed{
		Title:       fmt.Sprintf("Cartero Feed (%s)", f.name),
		Link:        &feeds.Link{Href: "http://localhost/"},
		Description: "Content aggregation feed from Cartero",
		Author:      &feeds.Author{Name: "Cartero"},
		Created:     time.Now().UTC(),
		Items:       items,
	}

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
	log.Printf("Feed target %s: Atom feed generated and cached (%d bytes)", f.name, len(atom))
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

	f.mu.RLock()
	items := make([]*feeds.Item, len(f.items))
	copy(items, f.items)
	totalItems := len(f.items)
	f.mu.RUnlock()

	log.Printf("Feed target %s: handleJSONFeed called. Total items in memory: %d", f.name, totalItems)

	sort.Slice(items, func(i, j int) bool {
		return items[i].Created.After(items[j].Created)
	})

	if len(items) > f.config.MaxItems {
		items = items[:f.config.MaxItems]
	}

	log.Printf("Feed target %s: serving %d items in JSON feed (max_items=%d)", f.name, len(items), f.config.MaxItems)

	feed := &feeds.Feed{
		Title:       fmt.Sprintf("Cartero Feed (%s)", f.name),
		Link:        &feeds.Link{Href: "http://localhost/"},
		Description: "Content aggregation feed from Cartero",
		Author:      &feeds.Author{Name: "Cartero"},
		Created:     time.Now().UTC(),
		Items:       items,
	}

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
	log.Printf("Feed target %s: JSON feed generated and cached (%d bytes)", f.name, len(jsonStr))
	fmt.Fprint(w, jsonStr)
}

func (f *Target) handleHealth(w http.ResponseWriter, r *http.Request) {
	f.mu.RLock()
	itemCount := len(f.items)
	f.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","name":"%s","items":%d,"time":"%s"}`,
		f.name, itemCount, time.Now().UTC().Format(time.RFC3339))
}
