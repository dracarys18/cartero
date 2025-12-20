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

	"github.com/bwmarrin/discordgo"
)

type DiscordPlatform struct {
	botToken string
	sleep    time.Duration
	session  *discordgo.Session
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

	platform := &DiscordPlatform{
		botToken: botToken,
		sleep:    sleep,
	}

	// Create discordgo session to appear online
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		log.Printf("Discord platform: failed to create session: %v", err)
		return platform
	}

	platform.session = session

	// Open websocket connection
	if err := session.Open(); err != nil {
		log.Printf("Discord platform: failed to open session: %v", err)
	} else {
		log.Printf("Discord platform: bot is now ONLINE")
	}

	return platform
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

	embed := d.buildEmbed(item)

	// Convert discordgo embed back to our format for the API call
	// (discordgo doesn't support forum thread creation yet)
	dgEmbed := DiscordEmbed{
		Title:       embed.Title,
		Description: embed.Description,
		URL:         embed.URL,
		Color:       embed.Color,
		Timestamp:   embed.Timestamp,
	}

	if embed.Footer != nil {
		dgEmbed.Footer = &DiscordEmbedFooter{Text: embed.Footer.Text}
	}

	for _, f := range embed.Fields {
		dgEmbed.Fields = append(dgEmbed.Fields, DiscordEmbedField{
			Name:   f.Name,
			Value:  f.Value,
			Inline: f.Inline,
		})
	}

	payload := CreateThreadPayload{
		Name:        title,
		AutoArchive: 1440,
		Message: DiscordMessage{
			Embeds: []DiscordEmbed{dgEmbed},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Use standard HTTP client for forum threads (not supported by discordgo)
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/threads", d.channelID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", d.platform.botToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DiscordBot (cartero, 1.0)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
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
	embed := d.buildEmbed(item)

	msg, err := d.platform.session.ChannelMessageSendEmbed(d.channelID, embed)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	time.Sleep(d.platform.sleep)

	return msg.ID, nil
}

func (d *DiscordTarget) buildMessage(item *core.ProcessedItem) DiscordMessage {
	// Execute template with item
	var buf bytes.Buffer
	if err := d.template.Execute(&buf, item.Original); err != nil {
		panic(fmt.Sprintf("Discord target %s: template execution error: %v", d.name, err))
	}

	// Parse JSON output from template
	var embed DiscordEmbed
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &embed); err != nil {
		log.Printf("Discord target %s: template output: %s", d.name, output)
		panic(fmt.Sprintf("Discord target %s: failed to parse template output as JSON: %v", d.name, err))
	}

	return DiscordMessage{
		Embeds: []DiscordEmbed{embed},
	}
}

func (d *DiscordTarget) buildEmbed(item *core.ProcessedItem) *discordgo.MessageEmbed {
	// Execute template with item
	var buf bytes.Buffer
	if err := d.template.Execute(&buf, item.Original); err != nil {
		panic(fmt.Sprintf("Discord target %s: template execution error: %v", d.name, err))
	}

	// Parse JSON output from template
	var embed DiscordEmbed
	output := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(output), &embed); err != nil {
		log.Printf("Discord target %s: template output: %s", d.name, output)
		panic(fmt.Sprintf("Discord target %s: failed to parse template output as JSON: %v", d.name, err))
	}

	// Convert to discordgo.MessageEmbed
	dgEmbed := &discordgo.MessageEmbed{
		Title:       embed.Title,
		Description: embed.Description,
		URL:         embed.URL,
		Color:       embed.Color,
		Timestamp:   embed.Timestamp,
	}

	// Convert fields
	for _, field := range embed.Fields {
		dgEmbed.Fields = append(dgEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   field.Name,
			Value:  field.Value,
			Inline: field.Inline,
		})
	}

	// Convert footer
	if embed.Footer != nil {
		dgEmbed.Footer = &discordgo.MessageEmbedFooter{
			Text: embed.Footer.Text,
		}
	}

	return dgEmbed
}

func (d *DiscordTarget) Shutdown(ctx context.Context) error {
	log.Printf("Discord target %s: shutting down", d.name)
	return nil
}

func (p *DiscordPlatform) Shutdown() {
	log.Printf("Discord platform: shutting down")

	// Close discordgo session
	if p.session != nil {
		p.session.Close()
	}
}
