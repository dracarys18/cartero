package platforms

import (
	"cartero/internal/config"
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramPlatform struct {
	botToken string
	bot      *tgbotapi.BotAPI
}

func NewTelegramPlatform(settings *config.TelegramPlatformSettings) (*TelegramPlatform, error) {
	if settings.BotToken == "" {
		return nil, fmt.Errorf("telegram platform: bot_token is required")
	}

	return &TelegramPlatform{
		botToken: settings.BotToken,
	}, nil
}

func (p *TelegramPlatform) Validate() error {
	return nil
}

func (p *TelegramPlatform) Initialize(_ context.Context) error {
	bot, err := tgbotapi.NewBotAPI(p.botToken)
	if err != nil {
		return fmt.Errorf("telegram platform: failed to create bot: %w", err)
	}

	p.bot = bot

	return nil
}

func (p *TelegramPlatform) Close(_ context.Context) error {
	return nil
}

func (p *TelegramPlatform) Bot() *tgbotapi.BotAPI {
	return p.bot
}
