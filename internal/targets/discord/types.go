package discord

import (
	"encoding/json"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Payload struct {
	Embeds []Embed `json:"embeds"`
}

type Embed struct {
	Title        string       `json:"title,omitempty"`
	Description  string       `json:"description,omitempty"`
	URL          string       `json:"url,omitempty"`
	Color        int          `json:"color,omitempty"`
	Timestamp    string       `json:"timestamp,omitempty"`
	ThumbnailURL string       `json:"thumbnail_url,omitempty"`
	Fields       []EmbedField `json:"fields,omitempty"`
	Footer       *EmbedFooter `json:"footer,omitempty"`
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

func (p *Payload) TryFrom(templateOutput []byte) error {
	if err := json.Unmarshal(templateOutput, p); err != nil {
		return fmt.Errorf("discord: failed to unmarshal template output to Payload: %w", err)
	}
	return nil
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

	dgEmbed.Thumbnail = &discordgo.MessageEmbedThumbnail{
		URL: e.ThumbnailURL,
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
