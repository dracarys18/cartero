package targets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"text/template"
	"time"

	"cartero/internal/core"
	"cartero/internal/utils"
)

type FeedItem struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Link        string                 `json:"link"`
	Description string                 `json:"description"`
	Content     string                 `json:"content"`
	Author      string                 `json:"author"`
	Source      string                 `json:"source"`
	PublishedAt time.Time              `json:"published_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type FeedConfig struct {
	Port     string
	FeedSize int
	MaxItems int
}

type FeedTarget struct {
	name         string
	config       FeedConfig
	items        []FeedItem
	mu           sync.RWMutex
	server       *http.Server
	stopCh       chan struct{}
	startedCh    chan error
	rssTemplate  *template.Template
	atomTemplate *template.Template
	jsonTemplate *template.Template
}

type RSSFeedData struct {
	Title         string
	Link          string
	Description   string
	LastBuildDate string
	Items         []RSSItem
}

type RSSItem struct {
	ID          string
	Title       string
	Link        string
	Description string
	Author      string
	Source      string
	PubDate     string
	Content     string
}

type AtomFeedData struct {
	Title    string
	Subtitle string
	Updated  string
	Link     string
	ID       string
	Author   string
	Items    []AtomItem
}

type AtomItem struct {
	Title     string
	ID        string
	Published string
	Updated   string
	Author    string
	Link      string
	Summary   string
	Content   string
	Source    string
}

type JSONFeedData struct {
	Title        string
	Description  string
	DateModified string
	Items        []JSONItem
}

type JSONItem struct {
	ID          string
	Title       string
	Link        string
	Description string
	Content     string
	Author      string
	Source      string
	PublishedAt string
}

func NewFeedTarget(name string, config FeedConfig) *FeedTarget {
	if config.Port == "" {
		config.Port = "8080"
	}
	if config.FeedSize == 0 {
		config.FeedSize = 100
	}
	if config.MaxItems == 0 {
		config.MaxItems = 50
	}

	rssTemplate, _ := utils.LoadTemplate("templates/feed.rss.tmpl")
	atomTemplate, _ := utils.LoadTemplate("templates/feed.atom.tmpl")
	jsonTemplate, _ := utils.LoadTemplate("templates/feed.json.tmpl")

	return &FeedTarget{
		name:         name,
		config:       config,
		items:        make([]FeedItem, 0, config.FeedSize),
		stopCh:       make(chan struct{}),
		startedCh:    make(chan error, 1),
		rssTemplate:  rssTemplate,
		atomTemplate: atomTemplate,
		jsonTemplate: jsonTemplate,
	}
}

func (f *FeedTarget) Name() string {
	return f.name
}

func (f *FeedTarget) Initialize(ctx context.Context) error {
	if f.rssTemplate == nil {
		tmpl, _ := utils.LoadTemplate("templates/feed.rss.tmpl")
		f.rssTemplate = tmpl
	}
	if f.atomTemplate == nil {
		tmpl, _ := utils.LoadTemplate("templates/feed.atom.tmpl")
		f.atomTemplate = tmpl
	}
	if f.jsonTemplate == nil {
		tmpl, _ := utils.LoadTemplate("templates/feed.json.tmpl")
		f.jsonTemplate = tmpl
	}

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
		if err := f.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Feed target %s: server error: %v", f.name, err)
			select {
			case f.startedCh <- err:
			default:
			}
		}
	}()

	time.Sleep(100 * time.Millisecond)
	return nil
}

func (f *FeedTarget) Publish(ctx context.Context, item *core.ProcessedItem) (*core.PublishResult, error) {
	feedItem := f.convertToFeedItem(item)

	f.mu.Lock()
	f.items = append(f.items, feedItem)
	if len(f.items) > f.config.FeedSize {
		f.items = f.items[len(f.items)-f.config.FeedSize:]
	}
	f.mu.Unlock()

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

func (f *FeedTarget) Shutdown(ctx context.Context) error {
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

func (f *FeedTarget) convertToFeedItem(item *core.ProcessedItem) FeedItem {
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

	return FeedItem{
		ID:          item.Original.ID,
		Title:       title,
		Link:        link,
		Description: description,
		Content:     content,
		Author:      author,
		Source:      item.Original.Source,
		PublishedAt: item.Original.Timestamp,
		Metadata:    item.Original.Metadata,
	}
}

func (f *FeedTarget) handleRSSFeed(w http.ResponseWriter, r *http.Request) {
	f.mu.RLock()
	items := make([]FeedItem, len(f.items))
	copy(items, f.items)
	f.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].PublishedAt.After(items[j].PublishedAt)
	})

	if len(items) > f.config.MaxItems {
		items = items[:f.config.MaxItems]
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	rssItems := make([]RSSItem, len(items))
	for i, item := range items {
		rssItems[i] = RSSItem{
			ID:          item.ID,
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Author:      item.Author,
			Source:      item.Source,
			PubDate:     item.PublishedAt.UTC().Format(time.RFC1123Z),
			Content:     item.Content,
		}
	}

	lastBuildDate := time.Now().UTC().Format(time.RFC1123Z)
	if len(items) > 0 {
		lastBuildDate = items[0].PublishedAt.UTC().Format(time.RFC1123Z)
	}

	feedData := RSSFeedData{
		Title:         fmt.Sprintf("Cartero Feed (%s)", f.name),
		Link:          "http://localhost/",
		Description:   "Content aggregation feed from Cartero",
		LastBuildDate: lastBuildDate,
		Items:         rssItems,
	}

	if f.rssTemplate == nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "RSS template not available")
		return
	}

	var buf bytes.Buffer
	if err := f.rssTemplate.Execute(&buf, feedData); err != nil {
		log.Printf("Feed target %s: RSS template error: %v", f.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error generating feed: %v", err)
		return
	}

	w.Write(buf.Bytes())
}

func (f *FeedTarget) handleAtomFeed(w http.ResponseWriter, r *http.Request) {
	f.mu.RLock()
	items := make([]FeedItem, len(f.items))
	copy(items, f.items)
	f.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].PublishedAt.After(items[j].PublishedAt)
	})

	if len(items) > f.config.MaxItems {
		items = items[:f.config.MaxItems]
	}

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	atomItems := make([]AtomItem, len(items))
	for i, item := range items {
		atomItems[i] = AtomItem{
			Title:     item.Title,
			ID:        item.ID,
			Published: item.PublishedAt.UTC().Format(time.RFC3339),
			Updated:   item.PublishedAt.UTC().Format(time.RFC3339),
			Author:    item.Author,
			Link:      item.Link,
			Summary:   item.Description,
			Content:   item.Content,
			Source:    item.Source,
		}
	}

	updated := time.Now().UTC().Format(time.RFC3339)
	if len(items) > 0 {
		updated = items[0].PublishedAt.UTC().Format(time.RFC3339)
	}

	feedData := AtomFeedData{
		Title:    fmt.Sprintf("Cartero Feed (%s)", f.name),
		Subtitle: "Content aggregation feed from Cartero",
		Updated:  updated,
		Link:     "http://localhost/",
		ID:       "urn:uuid:cartero-feed",
		Author:   "Cartero",
		Items:    atomItems,
	}

	if f.atomTemplate == nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Atom template not available")
		return
	}

	var buf bytes.Buffer
	if err := f.atomTemplate.Execute(&buf, feedData); err != nil {
		log.Printf("Feed target %s: Atom template error: %v", f.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error generating feed: %v", err)
		return
	}

	w.Write(buf.Bytes())
}

func (f *FeedTarget) handleJSONFeed(w http.ResponseWriter, r *http.Request) {
	f.mu.RLock()
	items := make([]FeedItem, len(f.items))
	copy(items, f.items)
	f.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].PublishedAt.After(items[j].PublishedAt)
	})

	if len(items) > f.config.MaxItems {
		items = items[:f.config.MaxItems]
	}

	w.Header().Set("Content-Type", "application/feed+json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	jsonItems := make([]JSONItem, len(items))
	for i, item := range items {
		jsonItems[i] = JSONItem{
			ID:          item.ID,
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Content:     item.Content,
			Author:      item.Author,
			Source:      item.Source,
			PublishedAt: item.PublishedAt.Format(time.RFC3339),
		}
	}

	feedData := JSONFeedData{
		Title:        fmt.Sprintf("Cartero Feed (%s)", f.name),
		Description:  "Content aggregation feed from Cartero",
		DateModified: time.Now().UTC().Format(time.RFC3339),
		Items:        jsonItems,
	}

	if f.jsonTemplate == nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "JSON template not available")
		return
	}

	var buf bytes.Buffer
	if err := f.jsonTemplate.Execute(&buf, feedData); err != nil {
		log.Printf("Feed target %s: JSON template error: %v", f.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error generating feed: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf.Bytes())
}

func (f *FeedTarget) handleHealth(w http.ResponseWriter, r *http.Request) {
	f.mu.RLock()
	itemCount := len(f.items)
	f.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"name":   f.name,
		"items":  itemCount,
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}
