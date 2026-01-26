package discord

import (
	"cartero/internal/types"
	"encoding/json"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Payload struct {
	Embeds []Embed `json:"embeds"`
}

type Embed struct {
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	URL         string       `json:"url,omitempty"`
	Color       int          `json:"color,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Footer      *EmbedFooter `json:"footer,omitempty"`
	Image       *EmbedImage  `json:"image,omitempty"`
	Thumbnail   *EmbedImage  `json:"thumbnail,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type EmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

type EmbedImage struct {
	URL string `json:"url"`
}

func (p *Payload) TryFrom(templateOutput []byte) error {
	if err := json.Unmarshal(templateOutput, p); err != nil {
		return fmt.Errorf("discord: failed to unmarshal template output to Payload: %w", err)
	}
	return nil
}

func (e *Embed) From(item *types.Item) {
	if title, ok := item.Metadata["title"].(string); ok {
		e.Title = title
	}

	if url, ok := item.Metadata["url"].(string); ok {
		e.URL = url
	}

	e.Color = 3447003

	e.Timestamp = item.Timestamp.Format("2006-01-02T15:04:05Z07:00")

	if score, ok := item.Metadata["score"]; ok {
		e.Fields = append(e.Fields, EmbedField{
			Name:   "Score",
			Value:  fmt.Sprintf("%v", score),
			Inline: true,
		})
	}

	if author, ok := item.Metadata["author"]; ok {
		e.Fields = append(e.Fields, EmbedField{
			Name:   "Author",
			Value:  fmt.Sprintf("%v", author),
			Inline: true,
		})
	}

	if comments, ok := item.Metadata["comments"].(string); ok && comments != "" {
		e.Fields = append(e.Fields, EmbedField{
			Name:  "Comments",
			Value: fmt.Sprintf("[View Discussion](%s)", comments),
		})
	}

	if item.TextContent != nil && item.TextContent.Description != "" {
		summary := item.TextContent.Description
		if len(summary) > 1024 {
			summary = summary[:1021] + "..."
		}
		e.Fields = append(e.Fields, EmbedField{
			Name:  "Summary",
			Value: summary,
		})
	}

	e.Footer = &EmbedFooter{
		Text: fmt.Sprintf("Source: %s", item.Source),
	}

	thumbnail := item.GetThumbnail()
	if thumbnail != "" {
		e.Thumbnail = &EmbedImage{
			URL: thumbnail,
		}
	}

	if item.TextContent != nil && item.TextContent.Image != "" && item.TextContent.Image != thumbnail {
		e.Image = &EmbedImage{
			URL: item.TextContent.Image,
		}
	}
}

func (e *Embed) Into() *discordgo.MessageEmbed {
	dgEmbed := &discordgo.MessageEmbed{
		Title:       e.Title,
		Description: e.Description,
		URL:         e.URL,
		Color:       e.Color,
		Timestamp:   e.Timestamp,
	}

	for _, field := range e.Fields {
		dgEmbed.Fields = append(dgEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   field.Name,
			Value:  field.Value,
			Inline: field.Inline,
		})
	}

	if e.Footer != nil {
		dgEmbed.Footer = &discordgo.MessageEmbedFooter{
			Text:    e.Footer.Text,
			IconURL: e.Footer.IconURL,
		}
	}

	if e.Image != nil {
		dgEmbed.Image = &discordgo.MessageEmbedImage{
			URL: e.Image.URL,
		}
	}

	if e.Thumbnail != nil {
		dgEmbed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: e.Thumbnail.URL,
		}
	}

	return dgEmbed
}

func (p *Payload) Into() []*discordgo.MessageEmbed {
	embeds := make([]*discordgo.MessageEmbed, len(p.Embeds))
	for i, embed := range p.Embeds {
		embeds[i] = embed.Into()
	}
	return embeds
}
