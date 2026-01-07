package feed

import (
	"context"
	"fmt"
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
	startCh   chan error
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
		startCh:   make(chan error, 1),
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/feed.rss", s.handleRSSFeed)
	mux.HandleFunc("/feed.atom", s.handleAtomFeed)
	mux.HandleFunc("/feed.json", s.handleJSONFeed)
	mux.HandleFunc("/feed.health", s.handleHealth)

	s.server = &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: mux,
	}

	go func() {
		err := s.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			s.startCh <- err
			fmt.Printf("Feed server error: %v\n", err)
		}
	}()

	select {
	case err := <-s.startCh:
		return fmt.Errorf("failed to start feed server on port %s: %w", s.config.Port, err)
	case <-time.After(1 * time.Second):
		return nil
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil && err != context.Canceled && err != http.ErrServerClosed {
			fmt.Printf("Feed server shutdown error: %v\n", err)
		}

		time.Sleep(2 * time.Second)
	}
	return nil
}

func (s *Server) handleRSSFeed(w http.ResponseWriter, r *http.Request) {
	entries, err := s.feedStore.ListRecentEntries(r.Context(), s.config.FeedSize)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}

	feed := s.buildFeed(entries)
	rss, err := feed.ToRss()
	if err != nil {
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
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}

	feed := s.buildFeed(entries)
	atom, err := feed.ToAtom()
	if err != nil {
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
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}

	feed := s.buildFeed(entries)
	jsonStr, err := feed.ToJSON()
	if err != nil {
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
