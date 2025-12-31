package config

import (
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Bot        BotConfig                  `toml:"bot"`
	Storage    StorageConfig              `toml:"storage"`
	Platforms  map[string]PlatformConfig  `toml:"platforms"`
	Sources    map[string]SourceConfig    `toml:"sources"`
	Processors map[string]ProcessorConfig `toml:"processors"`
	Targets    map[string]TargetConfig    `toml:"targets"`
}

type BotConfig struct {
	Name     string `toml:"name"`
	Interval string `toml:"interval"`
	RunOnce  bool   `toml:"run_once"`
	Sleep    string `toml:"sleep"`
}

type StorageConfig struct {
	Type string `toml:"type"`
	Path string `toml:"path"`
}

type PlatformConfig struct {
	Type     string           `toml:"type"`
	Enabled  bool             `toml:"enabled"`
	Sleep    string           `toml:"sleep"`
	Settings PlatformSettings `toml:"settings"`
}

type PlatformSettings struct {
	DiscordPlatformSettings
	OllamaPlatformSettings
}

type DiscordPlatformSettings struct {
	BotToken string `toml:"bot_token"`
	Timeout  string `toml:"timeout"`
}

type OllamaPlatformSettings struct {
	Model string `toml:"model"`
}

type SourceConfig struct {
	Type     string         `toml:"type"`
	Enabled  bool           `toml:"enabled"`
	Targets  []string       `toml:"targets"`
	Settings SourceSettings `toml:"settings"`
}

type SourceSettings struct {
	MaxItems int `toml:"max_items"`

	HackerNewsSettings
	RSSSettings
	LobstersSettings
	LessWrongSettings
}

type HackerNewsSettings struct {
	StoryType string `toml:"story_type"`
}

type RSSSettings struct {
	FeedURL string `toml:"feed_url"`
}

type LobstersSettings struct {
	SortBy            string   `toml:"sort_by"`
	IncludeCategories []string `toml:"include_categories"`
	ExcludeCategories []string `toml:"exclude_categories"`
}

type LessWrongSettings struct {
}

type ProcessorConfig struct {
	Type      string            `toml:"type"`
	Enabled   bool              `toml:"enabled"`
	DependsOn []string          `toml:"depends_on"`
	Settings  ProcessorSettings `toml:"settings"`
}

type ProcessorSettings struct {
	DedupeSettings
	ScoreFilterSettings
	KeywordFilterSettings
	RateLimitSettings
	TokenBucketSettings
	SummarySettings
	ExtractFieldsSettings
	TemplateSettings
	ContentDedupeSettings
}

type DedupeSettings struct {
	TTL string `toml:"ttl"`
}

type ContentDedupeSettings struct {
	Field string `toml:"field"`
}

type ScoreFilterSettings struct {
	MinScore int `toml:"min_score"`
}

type KeywordFilterSettings struct {
	Keywords []string `toml:"keywords"`
	Mode     string   `toml:"mode"`
}

type RateLimitSettings struct {
	Limit  int    `toml:"limit"`
	Window string `toml:"window"`
}

type TokenBucketSettings struct {
	Capacity   int    `toml:"capacity"`
	RefillRate string `toml:"refill_rate"`
}

type SummarySettings struct {
	Model string `toml:"model"`
}

type ExtractFieldsSettings struct {
	Fields []string `toml:"fields"`
}

type TemplateSettings struct {
	Template string `toml:"template"`
}

type TargetConfig struct {
	Type     string         `toml:"type"`
	Enabled  bool           `toml:"enabled"`
	Platform string         `toml:"platform"`
	Settings TargetSettings `toml:"settings"`
}

type TargetSettings struct {
	DiscordTargetSettings
	FeedTargetSettings
}

type DiscordTargetSettings struct {
	ChannelID   string `toml:"channel_id"`
	ChannelType string `toml:"channel_type"`
}

type FeedTargetSettings struct {
	Port     string `toml:"port"`
	FeedSize int    `toml:"feed_size"`
	MaxItems int    `toml:"max_items"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	if config.Bot.Name == "" {
		config.Bot.Name = "cartero"
	}

	if config.Bot.Interval == "" {
		config.Bot.Interval = "5m"
	}

	if _, err := time.ParseDuration(config.Bot.Interval); err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}

	if config.Bot.Sleep == "" {
		config.Bot.Sleep = "2s"
	}

	if _, err := time.ParseDuration(config.Bot.Sleep); err != nil {
		return fmt.Errorf("invalid sleep duration: %w", err)
	}

	if config.Storage.Type == "" {
		config.Storage.Type = "sqlite"
	}

	if config.Storage.Path == "" {
		config.Storage.Path = "./cartero.db"
	}

	return nil
}

// Helpers

func ParseDuration(d string, def time.Duration) time.Duration {
	if d == "" {
		return def
	}
	if parsed, err := time.ParseDuration(d); err == nil {
		return parsed
	}
	return def
}
