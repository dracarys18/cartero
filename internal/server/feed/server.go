package feed

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"cartero/internal/storage"

	"github.com/gorilla/feeds"
)

type Config struct {
	Port     string
	FeedSize int
	MaxItems int
}

type Server struct {
	name      string
	config    Config
	feedStore storage.FeedStore
	server    *http.Server
}

func New(name string, config Config, feedStore storage.FeedStore) *Server {
	if config.Port == "" {
		config.Port = "8080"
	}
	if config.FeedSize == 0 {
		config.FeedSize = 100
	}
	if config.MaxItems == 0 {
		config.MaxItems = 50
	}

	return &Server{
		name:      name,
		config:    config,
		feedStore: feedStore,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/feed.rss", s.handleRSSFeed)
	mux.HandleFunc("/feed.atom", s.handleAtomFeed)
	mux.HandleFunc("/feed.json", s.handleJSONFeed)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: mux,
	}

	go func() {
		log.Printf("Feed server %s: starting on port %s", s.name, s.config.Port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Feed server %s: error: %v", s.name, err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	log.Printf("Feed server %s: listening on http://localhost:%s", s.name, s.config.Port)
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Feed server %s: shutdown error: %v", s.name, err)
		}
	}
	return nil
}

func (s *Server) handleRSSFeed(w http.ResponseWriter, r *http.Request) {
	entries, err := s.feedStore.ListRecentEntries(r.Context(), s.config.FeedSize)
	if err != nil {
		log.Printf("Feed server %s: failed to list entries: %v", s.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}

	feed := s.buildFeed(entries)
	rss, err := feed.ToRss()
	if err != nil {
		log.Printf("Feed server %s: failed to generate RSS: %v", s.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprint(w, rss)
}

func (s *Server) handleAtomFeed(w http.ResponseWriter, r *http.Request) {
	entries, err := s.feedStore.ListRecentEntries(r.Context(), s.config.FeedSize)
	if err != nil {
		log.Printf("Feed server %s: failed to list entries: %v", s.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}

	feed := s.buildFeed(entries)
	atom, err := feed.ToAtom()
	if err != nil {
		log.Printf("Feed server %s: failed to generate Atom: %v", s.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprint(w, atom)
}

func (s *Server) handleJSONFeed(w http.ResponseWriter, r *http.Request) {
	entries, err := s.feedStore.ListRecentEntries(r.Context(), s.config.FeedSize)
	if err != nil {
		log.Printf("Feed server %s: failed to list entries: %v", s.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}

	feed := s.buildFeed(entries)
	jsonStr, err := feed.ToJSON()
	if err != nil {
		log.Printf("Feed server %s: failed to generate JSON: %v", s.name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/feed+json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprint(w, jsonStr)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","name":"%s","time":"%s"}`, s.name, time.Now().UTC().Format(time.RFC3339))
}

func (s *Server) buildFeed(entries []storage.FeedEntry) *feeds.Feed {
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

	if len(items) > s.config.MaxItems {
		items = items[:s.config.MaxItems]
	}

	return &feeds.Feed{
		Title:       fmt.Sprintf("Cartero Feed (%s)", s.name),
		Link:        &feeds.Link{Href: "http://localhost/"},
		Description: "Content aggregation feed from Cartero",
		Author:      &feeds.Author{Name: "Cartero"},
		Created:     time.Now().UTC(),
		Items:       items,
	}
}
