package platforms

import (
	"cartero/internal/config"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
)

type BlueskyPlatform struct {
	identifier string
	password   string
	client     *xrpc.Client
	mu         sync.RWMutex
}

func NewBlueskyPlatform(settings *config.BlueskyPlatformSettings) (*BlueskyPlatform, error) {
	if settings.Identifier == "" {
		return nil, fmt.Errorf("bluesky platform: identifier is required")
	}
	if settings.Password == "" {
		return nil, fmt.Errorf("bluesky platform: password is required")
	}

	return &BlueskyPlatform{
		identifier: settings.Identifier,
		password:   settings.Password,
	}, nil
}

func (p *BlueskyPlatform) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.createSession(ctx)
}

func (p *BlueskyPlatform) createSession(ctx context.Context) error {
	client := &xrpc.Client{
		Host: "https://bsky.social",
	}

	auth, err := atproto.ServerCreateSession(ctx, client, &atproto.ServerCreateSession_Input{
		Identifier: p.identifier,
		Password:   p.password,
	})
	if err != nil {
		return fmt.Errorf("failed to authenticate with bluesky: %w", err)
	}

	client.Auth = &xrpc.AuthInfo{
		AccessJwt:  auth.AccessJwt,
		RefreshJwt: auth.RefreshJwt,
		Handle:     auth.Handle,
		Did:        auth.Did,
	}

	p.client = client
	return nil
}

func (p *BlueskyPlatform) Do(ctx context.Context, fn func(c *xrpc.Client) error) error {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("bluesky platform not initialized")
	}

	err := fn(client)
	if err != nil && strings.Contains(err.Error(), "ExpiredToken") {
		p.mu.Lock()

		if p.client == client {
			if err := p.createSession(ctx); err != nil {
				p.mu.Unlock()
				return fmt.Errorf("failed to refresh session: %w", err)
			}
		}

		client = p.client
		p.mu.Unlock()

		return fn(client)
	}

	return err
}

func (p *BlueskyPlatform) Close(ctx context.Context) error {
	return nil
}

func (p *BlueskyPlatform) Validate() error {
	return nil
}
