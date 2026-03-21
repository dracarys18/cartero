package telegram

import (
	"bytes"
	"cartero/internal/components"
	"cartero/internal/platforms"
	"cartero/internal/types"
	"cartero/internal/utils"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"
)

type Target struct {
	name     string
	chatID   int64
	platform *platforms.TelegramPlatform
	template *template.Template
}

func New(name string, chatID int64, registry *components.Registry) *Target {
	platformCmp := registry.Get(components.PlatformComponentName).(*components.PlatformComponent)

	tmpl, err := utils.LoadTemplate("templates/telegram.tmpl")
	if err != nil {
		panic(err.Error())
	}

	return &Target{
		name:     name,
		chatID:   chatID,
		platform: platformCmp.Telegram(),
		template: tmpl,
	}
}

func (t *Target) Name() string {
	return t.name
}

func (t *Target) Initialize(_ context.Context) error {
	return nil
}

func (t *Target) Publish(_ context.Context, item *types.Item) (*types.PublishResult, error) {
	var buf bytes.Buffer
	if err := t.template.Execute(&buf, item); err != nil {
		return &types.PublishResult{
			Success:   false,
			Target:    t.name,
			ItemID:    item.ID,
			Timestamp: time.Now(),
			Error:     err,
		}, fmt.Errorf("telegram: template execution error: %w", err)
	}

	text := strings.TrimSpace(buf.String())
	msg := newMessage(t.chatID, text)

	sent, err := t.platform.Bot().Send(msg)
	if err != nil {
		return &types.PublishResult{
			Success:   false,
			Target:    t.name,
			ItemID:    item.ID,
			Timestamp: time.Now(),
			Error:     err,
		}, fmt.Errorf("telegram: failed to send message: %w", err)
	}

	return &types.PublishResult{
		Success:   true,
		Target:    t.name,
		ItemID:    item.ID,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"message_id": sent.MessageID,
			"chat_id":    t.chatID,
		},
	}, nil
}

func (t *Target) Shutdown(_ context.Context) error {
	return nil
}
