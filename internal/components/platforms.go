package components

import (
	"cartero/internal/config"
	"cartero/internal/platforms"
	"context"
	"fmt"
)

type PlatformComponent struct {
	config          map[string]config.PlatformConfig
	discordPlatform *platforms.DiscordPlatform
	blueskyPlatform *platforms.BlueskyPlatform
	ollamaPlatforms map[string]*platforms.OllamaPlatform
}

func NewPlatformComponent(config map[string]config.PlatformConfig) *PlatformComponent {
	return &PlatformComponent{
		config:          config,
		ollamaPlatforms: make(map[string]*platforms.OllamaPlatform),
	}
}

func (c *PlatformComponent) Name() string {
	return PlatformComponentName
}

func (c *PlatformComponent) Dependencies() []string {
	return []string{}
}

func (c *PlatformComponent) Validate() error {
	return nil
}

func (c *PlatformComponent) Initialize(ctx context.Context) error {
	if discordCfg, exists := c.config["discord"]; exists && discordCfg.Enabled {
		discord, err := platforms.NewDiscordPlatform(&discordCfg.Settings.DiscordPlatformSettings, discordCfg.Sleep)
		if err != nil {
			return fmt.Errorf("failed to create discord platform: %w", err)
		}
		if err := discord.Validate(); err != nil {
			return fmt.Errorf("discord platform validation failed: %w", err)
		}
		if err := discord.Initialize(ctx); err != nil {
			return fmt.Errorf("discord platform initialization failed: %w", err)
		}
		c.discordPlatform = discord
	}

	if blueskyCfg, exists := c.config["bluesky"]; exists && blueskyCfg.Enabled {
		bluesky, err := platforms.NewBlueskyPlatform(&blueskyCfg.Settings.BlueskyPlatformSettings)
		if err != nil {
			return fmt.Errorf("failed to create bluesky platform: %w", err)
		}
		if err := bluesky.Validate(); err != nil {
			return fmt.Errorf("bluesky platform validation failed: %w", err)
		}
		if err := bluesky.Initialize(ctx); err != nil {
			return fmt.Errorf("bluesky platform initialization failed: %w", err)
		}
		c.blueskyPlatform = bluesky
	}
	return nil
}

func (c *PlatformComponent) Close(ctx context.Context) error {
	if c.discordPlatform != nil {
		c.discordPlatform.Close(ctx)
	}
	if c.blueskyPlatform != nil {
		c.blueskyPlatform.Close(ctx)
	}
	return nil
}

func (c *PlatformComponent) Discord() *platforms.DiscordPlatform {
	return c.discordPlatform
}

func (c *PlatformComponent) Bluesky() *platforms.BlueskyPlatform {
	return c.blueskyPlatform
}

func (c *PlatformComponent) OllamaPlatform(model string) *platforms.OllamaPlatform {
	if platform, exists := c.ollamaPlatforms[model]; exists {
		return platform
	}
	platform := platforms.NewOllamaPlatform(model)
	c.ollamaPlatforms[model] = platform
	return platform
}
