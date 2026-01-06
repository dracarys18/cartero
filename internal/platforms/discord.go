package platforms

import (
	"cartero/internal/config"
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

type DiscordPlatform struct {
	botToken string
	sleep    time.Duration
	session  *discordgo.Session
}

func NewDiscordPlatform(settings *config.DiscordPlatformSettings, sleepStr string) (*DiscordPlatform, error) {
	if settings.BotToken == "" {
		return nil, fmt.Errorf("discord platform: bot_token is required")
	}

	sleep := 1 * time.Second
	if sleepStr != "" {
		if s, err := time.ParseDuration(sleepStr); err == nil {
			sleep = s
		}
	}

	return &DiscordPlatform{
		botToken: settings.BotToken,
		sleep:    sleep,
	}, nil
}

func (p *DiscordPlatform) Validate() error {
	return nil
}

func (p *DiscordPlatform) Initialize(ctx context.Context) error {
	if p.botToken == "" {
		return nil
	}

	session, err := discordgo.New("Bot " + p.botToken)
	if err != nil {
		return fmt.Errorf("failed to create discord session: %w", err)
	}

	p.session = session

	if err := session.Open(); err != nil {
		return fmt.Errorf("failed to open discord session: %w", err)
	}

	return nil
}

func (p *DiscordPlatform) Close(ctx context.Context) error {
	if p.session != nil {
		p.session.Close()
	}
	return nil
}

func (p *DiscordPlatform) Session() *discordgo.Session {
	return p.session
}

func (p *DiscordPlatform) BotToken() string {
	return p.botToken
}

func (p *DiscordPlatform) SleepDuration() time.Duration {
	return p.sleep
}
