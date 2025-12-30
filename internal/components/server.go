package components

import (
	"context"
	"fmt"
	"log"

	"cartero/internal/server/feed"
	"cartero/internal/storage"
)

type ServerConfig struct {
	Name     string
	Port     string
	FeedSize int
	MaxItems int
}

type ServerComponent struct {
	feedStore storage.FeedStore
	configs   []ServerConfig
	servers   map[string]*feed.Server
}

func NewServerComponent(feedStore storage.FeedStore) *ServerComponent {
	return &ServerComponent{
		feedStore: feedStore,
		configs:   make([]ServerConfig, 0),
		servers:   make(map[string]*feed.Server),
	}
}

func (c *ServerComponent) Name() string {
	return "servers"
}

func (c *ServerComponent) Register(cfg ServerConfig) {
	c.configs = append(c.configs, cfg)
}

func (c *ServerComponent) Validate() error {
	return nil
}

func (c *ServerComponent) Initialize(ctx context.Context) error {
	for _, cfg := range c.configs {
		if err := c.startServer(ctx, cfg); err != nil {
			return err
		}
	}
	return nil
}

func (c *ServerComponent) startServer(ctx context.Context, cfg ServerConfig) error {
	if _, exists := c.servers[cfg.Name]; exists {
		return nil
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.FeedSize == 0 {
		cfg.FeedSize = 100
	}
	if cfg.MaxItems == 0 {
		cfg.MaxItems = 50
	}

	server := feed.New(cfg.Name, feed.Config{
		Port:     cfg.Port,
		FeedSize: cfg.FeedSize,
		MaxItems: cfg.MaxItems,
	}, c.feedStore)

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("servers: failed to start feed server %s: %w", cfg.Name, err)
	}

	c.servers[cfg.Name] = server
	return nil
}

func (c *ServerComponent) Close(ctx context.Context) error {
	for name, server := range c.servers {
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("[Servers] Error shutting down server %s: %v", name, err)
		}
	}
	return nil
}

func (c *ServerComponent) Servers() map[string]*feed.Server {
	return c.servers
}
