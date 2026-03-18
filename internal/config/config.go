package config

import (
	"cartero/internal/utils/file"
	"cartero/internal/utils/keywords"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Bot        BotConfig                  `toml:"bot"`
	Storage    StorageConfig              `toml:"storage"`
	Redis      RedisConfig                `toml:"redis"`
	Platforms  map[string]PlatformConfig  `toml:"platforms"`
	Sources    map[string]SourceConfig    `toml:"sources"`
	Processors map[string]ProcessorConfig `toml:"processors"`
	Targets    map[string]TargetConfig    `toml:"targets"`
}

type RedisConfig struct {
	Addr         string `toml:"addr"`
	Password     string `toml:"password"`
	DB           int    `toml:"db"`
	StreamMaxLen int64  `toml:"stream_max_len"`
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
	BlueskyPlatformSettings
}

type DiscordPlatformSettings struct {
	BotToken string `toml:"bot_token"`
	Timeout  string `toml:"timeout"`
}

type OllamaPlatformSettings struct {
	Model          string `toml:"model"`
	EmbeddingModel string `toml:"embedding_model"`
}

type BlueskyPlatformSettings struct {
	Identifier string `toml:"identifier"`
	Password   string `toml:"password"`
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
	ScraperSettings
}

type HackerNewsSettings struct {
	StoryType string `toml:"story_type"`
}

type RSSSettings struct {
	FeedURL string     `toml:"feed_url"`
	From    FromSource `toml:"from"`
}

type FromSource struct {
	Type  string `toml:"type"`
	Kind  string `toml:"kind"`
	Value string `toml:"value"`
}

type LobstersSettings struct {
	SortBy            string   `toml:"sort_by"`
	IncludeCategories []string `toml:"include_categories"`
	ExcludeCategories []string `toml:"exclude_categories"`
}

type LessWrongSettings struct {
}

type ScraperSettings struct {
	ScraperType string         `toml:"scraper_type"`
	ScraperName string         `toml:"scraper_name"`
	ScriptPath  string         `toml:"script_path"`
	Config      map[string]any `toml:"config"`
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
	PublishedAtFilterSettings
	RateLimitSettings
	TokenBucketSettings
	SummarySettings
	ExtractFieldsSettings
	TemplateSettings
	ExtractTextSettings
	EmbedTextSettings
}

type DedupeSettings struct {
	TTL string `toml:"ttl"`
}

type ScoreFilterSettings struct {
	MinScore int `toml:"min_score"`
}

type KeywordFilterSettings struct {
	Keywords         []keywords.KeywordWithContext `toml:"keywords"`
	ExactKeyword     []string                      `toml:"exact_keywords"`
	KeywordsFile     string                        `toml:"keywords_file"`
	Mode             string                        `toml:"mode"`
	DensityThreshold float64                       `toml:"density_threshold"`
	TitleBypass      bool                          `toml:"title_bypass"`
	EmbedThreshold   float64                       `toml:"embed_threshold"`
}

type EmbedTextSettings struct {
	ChunkSize int `toml:"chunk_size"`
}

type PublishedAtFilterSettings struct {
	After  string `toml:"after"`
	Before string `toml:"before"`
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

type ExtractTextSettings struct {
	Limit            int `toml:"limit"`
	MinContentLength int `toml:"min_content_length"`
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
	BlueskyTargetSettings
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

type BlueskyTargetSettings struct {
	Languages []string `toml:"languages"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	data = []byte(os.Expand(string(data), os.Getenv))

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := loadKeywordFiles(&config); err != nil {
		return nil, fmt.Errorf("failed to load keywords file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

func loadKeywordFiles(config *Config) error {
	for name, proc := range config.Processors {
		if proc.Settings.KeywordsFile == "" {
			continue
		}

		file := file.NewFile(proc.Settings.KeywordsFile)
		data, err := file.Get()
		if err != nil {
			return fmt.Errorf("processor %q: %w", name, err)
		}

		var keywords []keywords.KeywordWithContext
		err = json.Unmarshal(data, &keywords)

		if err != nil {
			log.Fatalf("processor %q: failed to parse keywords file: %v", name, err)
		}

		settings := proc.Settings
		settings.Keywords = append(settings.Keywords, keywords...)
		proc.Settings = settings
		config.Processors[name] = proc
	}
	return nil
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

	if config.Redis.Addr == "" {
		config.Redis.Addr = "localhost:6379"
	}

	if config.Redis.StreamMaxLen == 0 {
		config.Redis.StreamMaxLen = 1000
	}

	return nil
}

func ParseDuration(d string, def time.Duration) time.Duration {
	if d == "" {
		return def
	}
	if parsed, err := time.ParseDuration(d); err == nil {
		return parsed
	}
	return def
}
