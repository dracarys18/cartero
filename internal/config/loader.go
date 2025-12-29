package config

import (
	"fmt"
	"log"
	"time"

	"cartero/internal/core"
	"cartero/internal/platforms"
	"cartero/internal/processors"
	"cartero/internal/sources"
	"cartero/internal/storage"
	"cartero/internal/targets"
)

type Platforms struct {
	Discord *targets.DiscordPlatform
}

type Loader struct {
	config    *Config
	platforms *Platforms
	targets   map[string]core.Target
}

func NewLoader(config *Config) *Loader {
	return &Loader{
		config:    config,
		platforms: &Platforms{},
		targets:   make(map[string]core.Target),
	}
}

func (l *Loader) InitializePlatforms() error {
	for name, platformCfg := range l.config.Platforms {
		switch platformCfg.Type {
		case "discord":
			if l.platforms.Discord != nil {
				return fmt.Errorf("discord platform already initialized")
			}
			botToken := GetString(platformCfg.Settings, "bot_token", "")
			if botToken == "" {
				return fmt.Errorf("bot_token is required for discord platform %s", name)
			}
			timeout := GetDuration(platformCfg.Settings, "timeout", 60*time.Second)
			sleep := 1 * time.Second
			if platformCfg.Sleep != "" {
				if parsed, err := time.ParseDuration(platformCfg.Sleep); err == nil {
					sleep = parsed
				}
			}
			l.platforms.Discord = targets.NewDiscordPlatform(botToken, timeout, sleep)
		default:
			return fmt.Errorf("unsupported platform type: %s", platformCfg.Type)
		}
	}
	return nil
}

func (l *Loader) CreateStorage() (core.Storage, error) {
	switch l.config.Storage.Type {
	case "sqlite":
		return storage.NewSQLiteStorage(l.config.Storage.Path)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", l.config.Storage.Type)
	}
}

func (l *Loader) CreateSources() ([]core.Source, error) {
	var sources []core.Source

	for name, cfg := range l.config.Sources {
		if !cfg.Enabled {
			continue
		}

		source, err := l.createSource(name, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create source %s: %w", name, err)
		}
		sources = append(sources, source)
	}

	return sources, nil
}

func (l *Loader) createSource(name string, cfg SourceConfig) (core.Source, error) {
	switch cfg.Type {
	case "hackernews":
		storyType := GetString(cfg.Settings, "story_type", "topstories")
		maxItems := GetInt(cfg.Settings, "max_items", 30)
		return sources.NewHackerNewsSource(name, storyType, maxItems), nil

	case "lobsters":
		maxItems := GetInt(cfg.Settings, "max_items", 50)
		sortBy := GetString(cfg.Settings, "sort_by", "hot")
		includeCategories := GetStringSlice(cfg.Settings, "include_categories")
		excludeCategories := GetStringSlice(cfg.Settings, "exclude_categories")
		return sources.NewLobstersSource(name, maxItems, sortBy, includeCategories, excludeCategories), nil

	case "lesswrong":
		maxItems := GetInt(cfg.Settings, "max_items", 20)
		return sources.NewLessWrongSource(name, maxItems), nil

	case "rss":
		feedURL := GetString(cfg.Settings, "feed_url", "")
		if feedURL == "" {
			return nil, fmt.Errorf("feed_url is required for RSS source")
		}
		maxItems := GetInt(cfg.Settings, "max_items", 50)
		return sources.NewRSSSource(name, feedURL, maxItems), nil

	default:
		return nil, fmt.Errorf("unsupported source type: %s", cfg.Type)
	}
}

func (l *Loader) CreateProcessors() (map[string]core.Processor, error) {
	processorsMap := make(map[string]core.Processor)

	for name, cfg := range l.config.Processors {
		if !cfg.Enabled {
			continue
		}

		processor, err := l.createProcessor(name, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create processor %s: %w", name, err)
		}

		processorsMap[name] = processor
	}

	return processorsMap, nil
}

func (l *Loader) createProcessor(name string, cfg ProcessorConfig) (core.Processor, error) {
	switch cfg.Type {
	case "summary":
		model := GetString(cfg.Settings, "model", "")
		ollamaClient := platforms.NewOllamaPlatform(model)
		return processors.NewSummaryProcessor(name, ollamaClient), nil

	case "extract_fields":
		fields := GetStringSlice(cfg.Settings, "fields")
		return processors.FieldExtractor(name, fields), nil

	case "template":
		template := GetString(cfg.Settings, "template", "")
		if template == "" {
			return nil, fmt.Errorf("template is required for template processor")
		}
		return processors.TemplateTransformer(name, template), nil

	case "enrich":
		enrichments := cfg.Settings
		return processors.EnrichTransformer(name, enrichments), nil

	case "filter_score":
		minScore := GetInt(cfg.Settings, "min_score", 0)
		return processors.MinScoreFilter(name, minScore), nil

	case "filter_keyword":
		keywords := GetStringSlice(cfg.Settings, "keywords")
		mode := GetString(cfg.Settings, "mode", "include")
		return processors.KeywordFilter(name, keywords, mode), nil

	case "dedupe":
		ttl := GetDuration(cfg.Settings, "ttl", 24*time.Hour)
		return processors.NewDedupeProcessor(name, ttl), nil

	case "dedupe_content":
		fieldName := GetString(cfg.Settings, "field", "")
		return processors.NewContentDedupeProcessor(name, fieldName), nil

	case "rate_limit":
		limit := GetInt(cfg.Settings, "limit", 10)
		window := GetDuration(cfg.Settings, "window", 1*time.Minute)
		return processors.NewRateLimitProcessor(name, limit, window), nil

	case "token_bucket":
		capacity := GetInt(cfg.Settings, "capacity", 10)
		refillRate := GetDuration(cfg.Settings, "refill_rate", 1*time.Second)
		return processors.NewTokenBucketProcessor(name, capacity, refillRate), nil

	default:
		return nil, fmt.Errorf("unsupported processor type: %s", cfg.Type)
	}
}

func (l *Loader) CreateTarget(name string) (core.Target, error) {
	if target, exists := l.targets[name]; exists {
		return target, nil
	}

	cfg, exists := l.config.Targets[name]
	if !exists {
		return nil, fmt.Errorf("target %s not found in config", name)
	}

	if !cfg.Enabled {
		return nil, fmt.Errorf("target %s is not enabled", name)
	}

	target, err := l.createTarget(name, cfg)
	if err != nil {
		return nil, err
	}

	l.targets[name] = target
	return target, nil
}

func (l *Loader) createTarget(name string, cfg TargetConfig) (core.Target, error) {
	switch cfg.Type {
	case "discord":
		if cfg.Platform == "" {
			return nil, fmt.Errorf("platform is required for discord target")
		}

		if l.platforms.Discord == nil {
			return nil, fmt.Errorf("discord platform not initialized")
		}

		channelID := GetString(cfg.Settings, "channel_id", "")
		if channelID == "" {
			return nil, fmt.Errorf("channel_id is required for discord target")
		}
		channelType := GetString(cfg.Settings, "channel_type", "text")

		return targets.NewDiscordTarget(name, targets.DiscordConfig{
			Platform:    l.platforms.Discord,
			ChannelID:   channelID,
			ChannelType: channelType,
		}), nil

	case "feed":
		port := GetString(cfg.Settings, "port", "8080")
		feedSize := GetInt(cfg.Settings, "feed_size", 100)
		maxItems := GetInt(cfg.Settings, "max_items", 50)

		return targets.NewFeedTarget(name, targets.FeedConfig{
			Port:     port,
			FeedSize: feedSize,
			MaxItems: maxItems,
		}), nil

	default:
		return nil, fmt.Errorf("unsupported target type: %s", cfg.Type)
	}
}

func (l *Loader) BuildPipeline() (*core.Pipeline, error) {
	if err := l.InitializePlatforms(); err != nil {
		return nil, fmt.Errorf("failed to initialize platforms: %w", err)
	}

	store, err := l.CreateStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	pipeline := core.NewPipeline(store)

	// Create and add all processors (unified handling - no separate filters)
	log.Printf("BuildPipeline: Creating processors...")
	processorsMap, err := l.CreateProcessors()
	if err != nil {
		return nil, fmt.Errorf("failed to create processors: %w", err)
	}
	log.Printf("BuildPipeline: Created %d processors", len(processorsMap))

	for name, processor := range processorsMap {
		cfg, exists := l.config.Processors[name]
		if !exists {
			continue
		}

		processorConfig := core.ProcessorConfig{
			Name:      name,
			Type:      cfg.Type,
			Enabled:   cfg.Enabled,
			DependsOn: cfg.DependsOn,
		}
		pipeline.AddProcessorWithConfig(processor, processorConfig)
	}

	for sourceName, sourceCfg := range l.config.Sources {
		if !sourceCfg.Enabled {
			continue
		}

		source, err := l.createSource(sourceName, sourceCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create source %s: %w", sourceName, err)
		}

		var routeTargets []core.Target
		if len(sourceCfg.Targets) == 0 {
			return nil, fmt.Errorf("source %s has no targets configured", sourceName)
		}

		for _, targetName := range sourceCfg.Targets {
			target, err := l.CreateTarget(targetName)
			if err != nil {
				return nil, fmt.Errorf("failed to create target %s for source %s: %w", targetName, sourceName, err)
			}
			routeTargets = append(routeTargets, target)
		}

		route := core.SourceRoute{
			Source:  source,
			Targets: routeTargets,
		}

		pipeline.AddRoute(route)
	}

	return pipeline, nil
}

func (l *Loader) BuildBot() (*core.Bot, error) {
	pipeline, err := l.BuildPipeline()
	if err != nil {
		return nil, err
	}

	interval, _ := time.ParseDuration(l.config.Bot.Interval)

	bot := core.NewBot(core.BotConfig{
		Name:     l.config.Bot.Name,
		Pipeline: pipeline,
		Interval: interval,
		RunOnce:  l.config.Bot.RunOnce,
	})

	return bot, nil
}

func (l *Loader) Shutdown() error {
	return nil
}

func LoadAndBuild(configPath string) (*core.Bot, error) {
	config, err := Load(configPath)
	if err != nil {
		return nil, err
	}

	loader := NewLoader(config)
	return loader.BuildBot()
}
