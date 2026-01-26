package discord

import (
	"bytes"
	"cartero/internal/components"
	"cartero/internal/platforms"
	"cartero/internal/types"
	"cartero/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Target struct {
	name        string
	platform    *platforms.DiscordPlatform
	channelID   string
	channelType string
	template    *template.Template
}

func New(name string, channelID, channelType string, registry *components.Registry) *Target {
	platformCmp := registry.Get(components.PlatformComponentName).(*components.PlatformComponent)
	templatePath := "templates/discord.tmpl"

	tmpl, err := utils.LoadTemplate(templatePath)
	if err != nil {
		panic(err.Error())
	}

	return &Target{
		name:        name,
		channelID:   channelID,
		channelType: channelType,
		template:    tmpl,
		platform:    platformCmp.Discord(),
	}
}

func (d *Target) Name() string {
	return d.name
}

func (d *Target) Initialize(ctx context.Context) error {
	return nil
}

func (d *Target) Publish(ctx context.Context, item *types.Item) (*types.PublishResult, error) {
	var messageID string
	var err error

	switch d.channelType {
	case "forum":
		messageID, err = d.createForumThread(item)
	case "text":
		messageID, err = d.sendMessage(item)
	default:
		return nil, fmt.Errorf("unsupported channel type: %s", d.channelType)
	}

	if err != nil {
		return &types.PublishResult{
			Success:   false,
			Target:    d.name,
			ItemID:    item.ID,
			Timestamp: time.Now(),
			Error:     err,
		}, err
	}

	return &types.PublishResult{
		Success:   true,
		Target:    d.name,
		ItemID:    item.ID,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"message_id": messageID,
			"channel_id": d.channelID,
		},
	}, nil
}

func (d *Target) createForumThread(item *types.Item) (string, error) {
	title := "Untitled"
	if t, ok := item.Metadata["title"].(string); ok {
		title = t
		if len(title) > 100 {
			title = title[:97] + "..."
		}
	}

	embed, err := d.buildEmbed(item)
	if err != nil {
		return "", fmt.Errorf("failed to build embed: %w", err)
	}

	thread, err := d.platform.Session().ForumThreadStartEmbed(d.channelID, title, 1440, embed)

	if err != nil {
		return "", fmt.Errorf("failed to create forum thread: %w", err)
	}

	return thread.ID, nil
}

func (d *Target) sendMessage(item *types.Item) (string, error) {
	embed, err := d.buildEmbed(item)
	if err != nil {
		return "", fmt.Errorf("failed to build embed: %w", err)
	}

	msg, err := d.platform.Session().ChannelMessageSendEmbed(d.channelID, embed)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	time.Sleep(d.platform.SleepDuration())

	return msg.ID, nil
}

func (d *Target) buildEmbed(item *types.Item) (*discordgo.MessageEmbed, error) {
	var buf bytes.Buffer
	if err := d.template.Execute(&buf, item); err != nil {
		return nil, fmt.Errorf("template execution error: %w", err)
	}

	var embed Embed
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &embed); err != nil {
		return nil, fmt.Errorf("failed to parse template output: %w", err)
	}

	dgEmbed := embed.Into()

	return dgEmbed, nil
}

func (d *Target) Shutdown(ctx context.Context) error {
	return nil
}
