package config

import (
	"cartero/internal/utils/file"
	"cartero/internal/utils/keywords"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	Interests  InterestConfig             `toml:"interests"`
	Prompts    Prompts                    `toml:"prompts"`
	Blocklist  BlocklistConfig            `toml:"blocklist"`
}

type InterestConfig struct {
	Keywords             []keywords.KeywordWithContext `toml:"keywords"`
	KeywordsFile         string                        `toml:"keywords_file"`
	OffTopicThreshold    float64                       `toml:"off_topic_threshold"`
	MinQuality           float64                       `toml:"min_quality"`
	MinQualityIncomplete float64                       `toml:"min_quality_incomplete"`
}

type Prompts struct {
	Topic   TopicPrompts   `toml:"topic"`
	Quality QualityPrompts `toml:"quality"`
	Flags   []FlagPrompt   `toml:"flags"`
}

type TopicPrompts struct {
	Instructions string `toml:"instructions"`
	None         string `toml:"none"`
}

type QualityPrompts struct {
	Instructions string   `toml:"instructions"`
	Incomplete   string   `toml:"incomplete"`
	Levels       []string `toml:"levels"`
}

type FlagPrompt struct {
	Name      string  `toml:"name"`
	Question  string  `toml:"question"`
	Threshold float64 `toml:"threshold"`
}

type BlocklistConfig struct {
	Domains     []string `toml:"domains"`
	DomainsFile string   `toml:"domains_file"`
}

type RedisConfig struct {
	Addr     string `toml:"addr"`
	Password string `toml:"password"`
	DB       int    `toml:"db"`
}

type BotConfig struct {
	Name     string `toml:"name"`
	Interval string `toml:"interval"`
	RunOnce  bool   `toml:"run_once"`
	Sleep    string `toml:"sleep"`
}

type StorageConfig struct {
	Type string `toml:"type"`
	DSN  string `toml:"dsn"`
}

type PlatformConfig struct {
	Type     string           `toml:"type"`
	Enabled  bool             `toml:"enabled"`
	Sleep    string           `toml:"sleep"`
	Settings PlatformSettings `toml:"settings"`
}

type PlatformSettings struct {
	DiscordPlatformSettings
	BlueskyPlatformSettings
	TelegramPlatformSettings
	APIPlatformSettings
	EmbedderPlatformSettings
}

type APIPlatformSettings struct {
	BaseURL string `toml:"base_url"`
	APIKey  string `toml:"api_key"`
	Model   string `toml:"model"`
}

type EmbedderPlatformSettings struct {
	EmbeddingModel string `toml:"embedding_model"`
}

type DiscordPlatformSettings struct {
	BotToken string `toml:"bot_token"`
}

type BlueskyPlatformSettings struct {
	Identifier string `toml:"identifier"`
	Password   string `toml:"password"`
}

type TelegramPlatformSettings struct {
	BotToken string `toml:"tg_bot_token"`
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
	PublishedAtFilterSettings
	SummarySettings
	ExtractFieldsSettings
	TemplateSettings
	ExtractTextSettings
}

type DedupeSettings struct {
	EmbedThreshold float64 `toml:"embed_threshold"`
	EmbedWindow    string  `toml:"embed_window"`
	TTL            string  `toml:"ttl"`
}

type ScoreFilterSettings struct {
	MinScore int `toml:"min_score"`
}

type PublishedAtFilterSettings struct {
	After  string `toml:"after"`
	Before string `toml:"before"`
}

type SummarySettings struct {
	Model string `toml:"model"`
}

type ExtractFieldsSettings struct {
	Fields []string `toml:"fields"`
}

type ExtractTextSettings struct {
	MinContentLength int    `toml:"min_content_length"`
	Concurrency      int    `toml:"concurrency"`
	TimeoutSeconds   int    `toml:"timeout_seconds"`
	ReaderURL        string `toml:"reader_url"`
	Resolver         string `toml:"resolver"`
}

type TemplateSettings struct {
	Template string `toml:"template"`
}

type TargetConfig struct {
	Type     string         `toml:"type"`
	Enabled  bool           `toml:"enabled"`
	Settings TargetSettings `toml:"settings"`
}

type TargetSettings struct {
	DiscordTargetSettings
	FeedTargetSettings
	BlueskyTargetSettings
	TelegramTargetSettings
}

type DiscordTargetSettings struct {
	ChannelID   string `toml:"channel_id"`
	ChannelType string `toml:"channel_type"`
}

type FeedTargetSettings struct {
	Port            string `toml:"port"`
	FeedSize        int    `toml:"feed_size"`
	MaxItems        int    `toml:"max_items"`
	SiteURL         string `toml:"site_url"`
	SiteName        string `toml:"site_name"`
	SiteDescription string `toml:"site_description"`
}

type BlueskyTargetSettings struct {
	Languages []string `toml:"languages"`
}

type TelegramTargetSettings struct {
	ChatID int64 `toml:"chat_id"`
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

	if err := loadInterests(&config); err != nil {
		return nil, fmt.Errorf("failed to load interests file: %w", err)
	}

	if err := loadBlocklist(&config); err != nil {
		return nil, fmt.Errorf("failed to load blocklist file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

func loadInterests(config *Config) error {
	if config.Interests.KeywordsFile == "" {
		return nil
	}

	f := file.NewFile(config.Interests.KeywordsFile)
	data, err := f.Get()
	if err != nil {
		return err
	}

	var loaded []keywords.KeywordWithContext
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("parse %s: %w", config.Interests.KeywordsFile, err)
	}

	config.Interests.Keywords = append(config.Interests.Keywords, loaded...)
	return nil
}

func validatePrompts(p *Prompts) error {
	for i := range p.Flags {
		if p.Flags[i].Threshold == 0 {
			p.Flags[i].Threshold = 0.7
		}
	}

	switch {
	case p.Topic.Instructions == "" || p.Topic.None == "":
		return fmt.Errorf("topic.instructions and topic.none are required")
	case p.Quality.Instructions == "":
		return fmt.Errorf("quality.instructions is required")
	case len(p.Quality.Levels) < 2 || len(p.Quality.Levels) > 10:
		return fmt.Errorf("quality.levels must have 2 to 10 entries, got %d", len(p.Quality.Levels))
	}

	seen := map[string]bool{"interest": true, "quality": true}
	for _, flag := range p.Flags {
		switch {
		case flag.Name == "" || flag.Question == "":
			return fmt.Errorf("flags: name and question are required")
		case seen[flag.Name]:
			return fmt.Errorf("flags: duplicate or reserved name %q", flag.Name)
		case flag.Threshold < 0 || flag.Threshold > 1:
			return fmt.Errorf("flags %q: threshold must be between 0 and 1", flag.Name)
		}
		seen[flag.Name] = true
	}
	return nil
}

func loadBlocklist(config *Config) error {
	if config.Blocklist.DomainsFile == "" {
		return nil
	}

	f := file.NewFile(config.Blocklist.DomainsFile)
	data, err := f.Get()
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		config.Blocklist.Domains = append(config.Blocklist.Domains, line)
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
		config.Storage.Type = "postgres"
	}

	if config.Storage.DSN == "" {
		return fmt.Errorf("storage.dsn is required")
	}

	if config.Redis.Addr == "" {
		config.Redis.Addr = "localhost:6379"
	}

	if config.Interests.OffTopicThreshold <= 0 {
		config.Interests.OffTopicThreshold = 0.4
	}

	if config.Interests.MinQuality <= 0 {
		config.Interests.MinQuality = 2.0
	}

	if config.Interests.MinQualityIncomplete <= 0 {
		config.Interests.MinQualityIncomplete = 1.4
	}

	if len(config.Interests.Keywords) > 0 {
		if !hasEnabledPlatform(config, "jev") {
			return fmt.Errorf("interests require an enabled platform of type \"jev\"")
		}
		if err := validatePrompts(&config.Prompts); err != nil {
			return fmt.Errorf("prompts: %w", err)
		}
	}

	return nil
}

func hasEnabledPlatform(config *Config, platformType string) bool {
	for _, p := range config.Platforms {
		if p.Enabled && p.Type == platformType {
			return true
		}
	}
	return false
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
