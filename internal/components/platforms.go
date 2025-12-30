package components

import (
	"context"
	"fmt"
	"time"

	"cartero/internal/platforms"
)

type PlatformConfig struct {
	Type     string                 `toml:"type"`
	Sleep    string                 `toml:"sleep"`
	Settings map[string]interface{} `toml:"settings"`
}

type PlatformComponent struct {
	config          map[string]PlatformConfig
	discordPlatform *platforms.DiscordPlatform
	ollamaPlatforms map[string]*platforms.OllamaPlatform
}

func NewPlatformComponent(config map[string]PlatformConfig) *PlatformComponent {
	return &PlatformComponent{
		config:          config,
		ollamaPlatforms: make(map[string]*platforms.OllamaPlatform),
	}
}

func (c *PlatformComponent) Name() string {
	return "platforms"
}

func (c *PlatformComponent) Validate() error {
	return nil
}

func (c *PlatformComponent) Initialize(ctx context.Context) error {
	if discordCfg, exists := c.config["discord"]; exists {
		settings := discordCfg.Settings
		if discordCfg.Sleep != "" {
			if parsed, err := time.ParseDuration(discordCfg.Sleep); err == nil {
				settings["sleep"] = parsed
			}
		}

		discord, err := platforms.NewDiscordPlatform(settings)
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
	return nil
}

func (c *PlatformComponent) Close(ctx context.Context) error {
	if c.discordPlatform != nil {
		return c.discordPlatform.Close(ctx)
	}
	return nil
}

func (c *PlatformComponent) Discord() *platforms.DiscordPlatform {
	return c.discordPlatform
}

func (c *PlatformComponent) OllamaPlatform(model string) *platforms.OllamaPlatform {
	if platform, exists := c.ollamaPlatforms[model]; exists {
		return platform
	}
	platform := platforms.NewOllamaPlatform(model)
	c.ollamaPlatforms[model] = platform
	return platform
}
