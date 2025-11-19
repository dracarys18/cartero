package targets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"cartero/internal/core"
	"cartero/internal/utils"
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
	template    *template.Template
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
	// Use platform name for template selection
	templatePath := "templates/discord.tmpl"

	// Load template from file
	tmpl, err := utils.LoadTemplate(templatePath)
	if err != nil {
		log.Printf("Discord target %s: FATAL - %v", name, err)
		panic(err.Error())
	}

	log.Printf("Discord target %s: loaded template from %s", name, templatePath)

	return &DiscordTarget{
		name:        name,
		platform:    config.Platform,
		channelID:   config.ChannelID,
		channelType: config.ChannelType,
		template:    tmpl,
	}
}

func (d *DiscordTarget) Name() string {
	return d.name
}

func (d *DiscordTarget) Initialize(ctx context.Context) error {
	log.Printf("Discord target %s: initializing (channel_id=%s, type=%s)", d.name, d.channelID, d.channelType)
	log.Printf("Discord target %s: initialization complete", d.name)
	return nil
}

func (d *DiscordTarget) Publish(ctx context.Context, item *core.ProcessedItem) (*core.PublishResult, error) {
	log.Printf("Discord target %s: publishing item %s to %s channel %s",
		d.name, item.Original.ID, d.channelType, d.channelID)

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
		return &core.PublishResult{
			Success:   false,
			Target:    d.name,
			ItemID:    item.Original.ID,
			Timestamp: time.Now(),
			Error:     err,
		}, err
	}

	return &core.PublishResult{
		Success:   true,
		Target:    d.name,
		ItemID:    item.Original.ID,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"message_id": messageID,
			"channel_id": d.channelID,
		},
	}, nil
}

func (d *DiscordTarget) createForumThread(item *core.ProcessedItem) (string, error) {
	title := "Untitled"
	if t, ok := item.Original.Metadata["title"].(string); ok {
		title = t
		if len(title) > 100 {
			title = title[:97] + "..."
		}
	}

	log.Printf("Discord target %s: creating forum thread '%s'", d.name, title)

	message := d.buildMessage(item)

	payload := CreateThreadPayload{
		Name:        title,
		AutoArchive: 1440,
		Message:     message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/threads", d.channelID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", d.platform.botToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DiscordBot (cartero, 1.0)")

	resp, err := d.platform.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 429 {
		var errorResp DiscordErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			log.Printf("Discord target %s: rate limited creating thread, retry_after=%.2fs", d.name, errorResp.RetryAfter)
			return "", fmt.Errorf("discord API rate limited, body: %s", string(respBody))
		}
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var thread DiscordThread
	if err := json.Unmarshal(respBody, &thread); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	log.Printf("Discord target %s: forum thread created successfully in channel %s", d.name, d.channelID)

	time.Sleep(d.platform.sleep)

	return thread.ID, nil
}

func (d *DiscordTarget) sendMessage(item *core.ProcessedItem) (string, error) {
	message := d.buildMessage(item)

	body, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", d.channelID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", d.platform.botToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DiscordBot (cartero, 1.0)")

	resp, err := d.platform.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 429 {
		var errorResp DiscordErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			log.Printf("Discord target %s: rate limited, retry_after=%.2fs", d.name, errorResp.RetryAfter)
			return "", fmt.Errorf("discord API rate limited, body: %s", string(respBody))
		}
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	time.Sleep(d.platform.sleep)

	return result.ID, nil
}

func (d *DiscordTarget) buildMessage(item *core.ProcessedItem) DiscordMessage {
	if str, ok := item.Data.(string); ok {
		return DiscordMessage{
			Content: str,
		}
	}

	// Prepare data for template
	data := make(map[string]interface{})
	for k, v := range item.Original.Metadata {
		data[k] = v
	}
	data["source"] = item.Original.Source
	data["timestamp"] = item.Original.Timestamp.Format(time.RFC3339)

	// Ensure description is not too long
	if desc, ok := data["description"].(string); ok && len(desc) > 2048 {
		data["description"] = desc[:2045] + "..."
	}

	// Execute template
	var buf bytes.Buffer
	if err := d.template.Execute(&buf, data); err != nil {
		log.Printf("Discord target %s: template execution error: %v", d.name, err)
		// Fallback to basic embed
		return DiscordMessage{
			Embeds: []DiscordEmbed{{
				Title:       fmt.Sprintf("%v", data["title"]),
				URL:         fmt.Sprintf("%v", data["url"]),
				Description: fmt.Sprintf("%v", data["description"]),
				Color:       3447003,
			}},
		}
	}

	// Parse JSON output from template
	var embed DiscordEmbed
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &embed); err != nil {
		log.Printf("Discord target %s: failed to parse template output as JSON: %v", d.name, err)
		log.Printf("Template output: %s", output)
		// Fallback to basic embed
		return DiscordMessage{
			Embeds: []DiscordEmbed{{
				Title:       fmt.Sprintf("%v", data["title"]),
				URL:         fmt.Sprintf("%v", data["url"]),
				Description: fmt.Sprintf("%v", data["description"]),
				Color:       3447003,
			}},
		}
	}

	return DiscordMessage{
		Embeds: []DiscordEmbed{embed},
	}
}

func (d *DiscordTarget) Shutdown(ctx context.Context) error {
	log.Printf("Discord target %s: shutting down", d.name)
	return nil
}

func (p *DiscordPlatform) Shutdown() {
	log.Printf("Discord platform: closing idle connections")
	p.httpClient.CloseIdleConnections()
}
