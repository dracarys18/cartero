package platforms

import (
	"cartero/internal/config"
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
)

type BlueskyPlatform struct {
	identifier string
	password   string
	client     *xrpc.Client
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
	// Create a new XRPC client pointing to the Bluesky social PDS
	client := &xrpc.Client{
		Host: "https://bsky.social",
	}

	// Attempt to create a session (login)
	auth, err := atproto.ServerCreateSession(ctx, client, &atproto.ServerCreateSession_Input{
		Identifier: p.identifier,
		Password:   p.password,
	})
	if err != nil {
		return fmt.Errorf("failed to authenticate with bluesky: %w", err)
	}

	// Attach authentication info to the client for future requests
	client.Auth = &xrpc.AuthInfo{
		AccessJwt:  auth.AccessJwt,
		RefreshJwt: auth.RefreshJwt,
		Handle:     auth.Handle,
		Did:        auth.Did,
	}

	p.client = client

	return nil
}

func (p *BlueskyPlatform) Client() *xrpc.Client {
	return p.client
}

func (p *BlueskyPlatform) Close(ctx context.Context) error {
	// No specific cleanup needed for the stateless HTTP client
	return nil
}

func (p *BlueskyPlatform) Validate() error {
	return nil
}
