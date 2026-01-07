package components

import (
	"context"
	"fmt"

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
	registry *Registry
	configs  []ServerConfig
	servers  map[string]*feed.Server
}

func NewServerComponent(registry *Registry) *ServerComponent {
	return &ServerComponent{
		registry: registry,
		configs:  make([]ServerConfig, 0),
		servers:  make(map[string]*feed.Server),
	}
}

func (c *ServerComponent) Name() string {
	return ServerComponentName
}

func (c *ServerComponent) Dependencies() []string {
	return []string{StorageComponentName}
}

func (c *ServerComponent) Register(cfg ServerConfig) {
	c.configs = append(c.configs, cfg)
}

func (c *ServerComponent) Validate() error {
	return nil
}

func (c *ServerComponent) Initialize(ctx context.Context) error {
	storageComp := c.registry.Get(StorageComponentName).(*StorageComponent)
	feedStore := storageComp.Store().Feed()

	for _, cfg := range c.configs {
		if err := c.startServer(ctx, cfg, feedStore); err != nil {
			return err
		}
	}
	return nil
}

func (c *ServerComponent) startServer(ctx context.Context, cfg ServerConfig, feedStore storage.FeedStore) error {
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
	}, feedStore)

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("servers: failed to start feed server %s: %w", cfg.Name, err)
	}

	c.servers[cfg.Name] = server
	return nil
}

func (c *ServerComponent) Close(ctx context.Context) error {
	for _, server := range c.servers {
		if err := server.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *ServerComponent) Servers() map[string]*feed.Server {
	return c.servers
}
