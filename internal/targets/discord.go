package targets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cartero/internal/core"
)

type DiscordPlatform struct {
	botToken   string
	httpClient *http.Client
	sleep      time.Duration
}

type DiscordTarget struct {
	name        string
	platform    *DiscordPlatform
	channelID   string
	channelType string
}

type DiscordConfig struct {
	Platform    *DiscordPlatform
	ChannelID   string
	ChannelType string
}

type DiscordChannel struct {
	ID   string `json:"id"`
	Type int    `json:"type"`
}

type DiscordThread struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CreateThreadPayload struct {
	Name        string         `json:"name"`
	AutoArchive int            `json:"auto_archive_duration"`
	Message     DiscordMessage `json:"message"`
}

type DiscordMessage struct {
	Content string         `json:"content,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	URL         string              `json:"url,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type DiscordEmbedFooter struct {
	Text string `json:"text"`
}

type DiscordErrorResponse struct {
	Message    string  `json:"message"`
	RetryAfter float64 `json:"retry_after"`
	Global     bool    `json:"global"`
}

func NewDiscordPlatform(botToken string, timeout time.Duration, sleep time.Duration) *DiscordPlatform {
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	if sleep == 0 {
		sleep = 1 * time.Second
	}

	return &DiscordPlatform{
		botToken: botToken,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		sleep: sleep,
	}
}

func NewDiscordTarget(name string, config DiscordConfig) *DiscordTarget {
	return &DiscordTarget{
		name:        name,
		platform:    config.Platform,
		channelID:   config.ChannelID,
		channelType: config.ChannelType,
	}
}

func (d *DiscordTarget) Name() string {
	return d.name
}

func (d *DiscordTarget) Initialize(ctx context.Context) error {
	if d.platform == nil {
		return fmt.Errorf("platform is required")
	}
	if d.channelID == "" {
		return fmt.Errorf("channel ID is required")
	}

	if d.channelType == "" {
		d.channelType = "text"
	}

	return nil
}

func (d *DiscordTarget) Sleep(ctx context.Context) error {
	if d.platform.sleep > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d.platform.sleep):
			return nil
		}
	}
	return nil
}

func (d *DiscordTarget) Publish(ctx context.Context, item *core.ProcessedItem) (*core.PublishResult, error) {
	result := &core.PublishResult{
		Success:   false,
		Target:    d.name,
		ItemID:    item.Original.ID,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	message := d.buildMessage(item)

	switch d.channelType {
	case "forum":
		return d.publishToForum(ctx, item, message, result)
	case "text":
		return d.publishToChannel(ctx, message, result)
	default:
		return nil, nil
	}
}

func (d *DiscordTarget) publishToChannel(ctx context.Context, message DiscordMessage, result *core.PublishResult) (*core.PublishResult, error) {
	jsonData, err := json.Marshal(message)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal message: %w", err)
		return result, result.Error
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", d.channelID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		return result, result.Error
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", d.platform.botToken))

	resp, err := d.platform.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to send request: %w", err)
		return result, result.Error
	}
	defer resp.Body.Close()

	result.Metadata["status_code"] = resp.StatusCode
	result.Metadata["channel_id"] = d.channelID

	if resp.StatusCode == 429 {
		body, _ := io.ReadAll(resp.Body)
		var errorResp DiscordErrorResponse
		if json.Unmarshal(body, &errorResp) == nil && errorResp.RetryAfter > 0 {
			result.Metadata["retry_after"] = errorResp.RetryAfter
		}
		result.Error = fmt.Errorf("discord API rate limited, body: %s", string(body))
		return result, result.Error
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Errorf("discord API returned status code: %d, body: %s", resp.StatusCode, string(body))
		return result, result.Error
	}

	result.Success = true
	return result, nil
}

func (d *DiscordTarget) publishToForum(ctx context.Context, item *core.ProcessedItem, message DiscordMessage, result *core.PublishResult) (*core.PublishResult, error) {
	threadName := "New Post"
	if title, ok := item.Original.Metadata["title"].(string); ok {
		threadName = title
		if len(threadName) > 100 {
			threadName = threadName[:97] + "..."
		}
	}

	threadPayload := CreateThreadPayload{
		Name:        threadName,
		AutoArchive: 60,
		Message:     message,
	}

	jsonData, err := json.Marshal(threadPayload)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal thread payload: %w", err)
		return result, result.Error
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/threads", d.channelID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		return result, result.Error
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", d.platform.botToken))

	resp, err := d.platform.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to send request: %w", err)
		return result, result.Error
	}
	defer resp.Body.Close()

	result.Metadata["status_code"] = resp.StatusCode
	result.Metadata["channel_id"] = d.channelID
	result.Metadata["is_forum"] = true

	if resp.StatusCode == 429 {
		body, _ := io.ReadAll(resp.Body)
		var errorResp DiscordErrorResponse
		if json.Unmarshal(body, &errorResp) == nil && errorResp.RetryAfter > 0 {
			result.Metadata["retry_after"] = errorResp.RetryAfter
		}
		result.Error = fmt.Errorf("discord API rate limited, body: %s", string(body))
		return result, result.Error
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Errorf("discord API returned status code: %d, body: %s", resp.StatusCode, string(body))
		return result, result.Error
	}

	result.Success = true
	return result, nil
}

func (d *DiscordTarget) buildMessage(item *core.ProcessedItem) DiscordMessage {
	if str, ok := item.Data.(string); ok {
		return DiscordMessage{
			Content: str,
		}
	}

	embed := DiscordEmbed{
		Color:     3447003,
		Timestamp: item.Original.Timestamp.Format(time.RFC3339),
		Footer: &DiscordEmbedFooter{
			Text: fmt.Sprintf("Source: %s", item.Original.Source),
		},
	}

	if title, ok := item.Original.Metadata["title"].(string); ok {
		embed.Title = title
	}

	if url, ok := item.Original.Metadata["url"].(string); ok {
		embed.URL = url
	}

	if description, ok := item.Original.Metadata["description"].(string); ok {
		if len(description) > 2048 {
			description = description[:2045] + "..."
		}
		embed.Description = description
	}

	if score, ok := item.Original.Metadata["score"].(int); ok {
		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:   "Score",
			Value:  fmt.Sprintf("%d", score),
			Inline: true,
		})
	}

	if author, ok := item.Original.Metadata["author"].(string); ok {
		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:   "Author",
			Value:  author,
			Inline: true,
		})
	}

	if comments, ok := item.Original.Metadata["comments"].(string); ok {
		if comments != "" {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   "Comments",
				Value:  fmt.Sprintf("[Discussion](%s)", comments),
				Inline: false,
			})
		}
	}

	return DiscordMessage{
		Embeds: []DiscordEmbed{embed},
	}
}

func (d *DiscordTarget) Shutdown(ctx context.Context) error {
	return nil
}

func (p *DiscordPlatform) Shutdown() {
	p.httpClient.CloseIdleConnections()
}
