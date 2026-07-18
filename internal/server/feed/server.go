package feed

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"cartero/internal/platforms"
	"cartero/internal/server/feed/handler"
	"cartero/internal/storage"
)

type Config struct {
	Port              string
	FeedSize          int
	MaxItems          int
	SiteURL           string
	SiteName          string
	SiteDescription   string
	SearchMaxDistance float64
}

type Server struct {
	name    string
	config  Config
	handler *handler.Handler
	server  *http.Server
	startCh chan error
}

func New(name string, config Config, entryStore storage.EntryStore, embedder platforms.Embedder) *Server {
	if config.Port == "" {
		config.Port = "8080"
	}
	if config.FeedSize == 0 {
		config.FeedSize = 100
	}
	if config.MaxItems == 0 {
		config.MaxItems = 50
	}

	h := handler.New(handler.Config{
		Name:              name,
		FeedSize:          config.FeedSize,
		MaxItems:          config.MaxItems,
		SiteURL:           config.SiteURL,
		SiteName:          config.SiteName,
		SiteDescription:   config.SiteDescription,
		SearchMaxDistance: config.SearchMaxDistance,
	}, entryStore, embedder)

	return &Server{
		name:    name,
		config:  config,
		handler: h,
		startCh: make(chan error, 1),
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.server = &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: s.handler.Routes(),
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
	}
	return nil
}
