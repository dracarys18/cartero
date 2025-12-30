package config

import (
	"context"
	"fmt"
	"log"
	"time"

	"cartero/internal/components"
	"cartero/internal/core"
	"cartero/internal/platforms"
	"cartero/internal/processors"
	"cartero/internal/sources"
	"cartero/internal/storage"
	"cartero/internal/targets"
)

type Loader struct {
	config       *Config
	storageComp  *components.StorageComponent
	platformComp *components.PlatformComponent
	serverComp   *components.ServerComponent
	pipeline     *core.Pipeline
}

func NewLoader(cfg *Config) *Loader {
	return &Loader{
		config: cfg,
	}
}

func (l *Loader) Initialize(ctx context.Context) (*core.Bot, error) {
	if err := l.initializeComponents(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	if l.pipeline == nil {
		return nil, fmt.Errorf("pipeline not initialized")
	}

	interval, err := time.ParseDuration(l.config.Bot.Interval)
	if err != nil {
		return nil, fmt.Errorf("invalid bot interval: %w", err)
	}

	shutdownFn := func() error {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return l.Shutdown(shutdownCtx)
	}

	bot := core.NewBot(core.BotConfig{
		Name:       l.config.Bot.Name,
		Pipeline:   l.pipeline,
		Interval:   interval,
		RunOnce:    l.config.Bot.RunOnce,
		ShutdownFn: shutdownFn,
	})

	return bot, nil
}

func (l *Loader) initializeComponents(ctx context.Context) error {
	log.Printf("[Loader] Initializing all components")

	storage := components.NewStorageComponent(l.config.Storage.Path)

	if err := storage.Validate(); err != nil {
		return fmt.Errorf("storage component validation failed: %w", err)
	}

	if err := storage.Initialize(ctx); err != nil {
		return fmt.Errorf("storage component initialization failed: %w", err)
	}

	l.storageComp = storage
	platform := l.buildPlatformComponent()

	if err := platform.Validate(); err != nil {
		return fmt.Errorf("platform component validation failed: %w", err)
	}

	if err := platform.Initialize(ctx); err != nil {
		return fmt.Errorf("platform component initialization failed: %w", err)
	}

	l.platformComp = platform
	store := l.storageComp.Store()
	l.serverComp = components.NewServerComponent(store.Feed())

	for name, targetCfg := range l.config.Targets {
		if targetCfg.Type != "feed" {
			continue
		}

		port := GetString(targetCfg.Settings, "port", "8080")
		feedSize := GetInt(targetCfg.Settings, "feed_size", 100)
		maxItems := GetInt(targetCfg.Settings, "max_items", 50)

		l.serverComp.Register(components.ServerConfig{
			Name:     name,
			Port:     port,
			FeedSize: feedSize,
			MaxItems: maxItems,
		})
	}

	if err := l.serverComp.Validate(); err != nil {
		return fmt.Errorf("server component validation failed: %w", err)
	}

	if err := l.serverComp.Initialize(ctx); err != nil {
		return fmt.Errorf("server component initialization failed: %w", err)
	}

	if err := l.buildPipeline(ctx); err != nil {
		return fmt.Errorf("failed to build pipeline: %w", err)
	}

	log.Printf("[Loader] All components initialized successfully")
	return nil
}

func (l *Loader) buildPlatformComponent() *components.PlatformComponent {
	platformsConfig := make(map[string]components.PlatformConfig)
	for name, cfg := range l.config.Platforms {
		platformsConfig[name] = components.PlatformConfig{
			Type:     cfg.Type,
			Sleep:    cfg.Sleep,
			Settings: cfg.Settings,
		}
	}
	return components.NewPlatformComponent(platformsConfig)
}

func (l *Loader) buildPipeline(ctx context.Context) error {
	store := l.storageComp.Store()
	l.pipeline = core.NewPipeline(store.Items())

	if err := l.addProcessors(l.pipeline); err != nil {
		return fmt.Errorf("failed to add processors: %w", err)
	}

	if err := l.addSourceRoutes(l.pipeline); err != nil {
		return fmt.Errorf("failed to add source routes: %w", err)
	}

	return nil
}

func (l *Loader) addProcessors(pipeline *core.Pipeline) error {
	for name, processorCfg := range l.config.Processors {
		if !processorCfg.Enabled {
			continue
		}

		processor, err := l.createProcessor(name, processorCfg)
		if err != nil {
			return fmt.Errorf("failed to create processor %s: %w", name, err)
		}

		pipelineConfig := core.ProcessorConfig{
			Name:      name,
			Type:      processorCfg.Type,
			Enabled:   processorCfg.Enabled,
			DependsOn: processorCfg.DependsOn,
		}
		pipeline.AddProcessorWithConfig(processor, pipelineConfig)
	}

	return nil
}

func (l *Loader) addSourceRoutes(pipeline *core.Pipeline) error {
	store := l.storageComp.Store()

	for sourceName, sourceCfg := range l.config.Sources {
		if !sourceCfg.Enabled {
			continue
		}

		source, err := l.createSource(sourceName, sourceCfg)
		if err != nil {
			return fmt.Errorf("failed to create source %s: %w", sourceName, err)
		}

		var routeTargets []core.Target
		if len(sourceCfg.Targets) == 0 {
			return fmt.Errorf("source %s has no targets configured", sourceName)
		}

		for _, targetName := range sourceCfg.Targets {
			targetCfg, exists := l.config.Targets[targetName]
			if !exists {
				return fmt.Errorf("target %s not found in config for source %s", targetName, sourceName)
			}
			if !targetCfg.Enabled {
				continue
			}

			target, err := l.createTarget(targetName, targetCfg, store)
			if err != nil {
				return fmt.Errorf("failed to create target %s for source %s: %w", targetName, sourceName, err)
			}
			routeTargets = append(routeTargets, target)
		}

		if len(routeTargets) == 0 {
			return fmt.Errorf("source %s has no enabled targets", sourceName)
		}

		route := core.SourceRoute{
			Source:  source,
			Targets: routeTargets,
		}
		pipeline.AddRoute(route)
	}

	return nil
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

func (l *Loader) createProcessor(name string, cfg ProcessorConfig) (core.Processor, error) {
	switch cfg.Type {
	case "summary":
		model := GetString(cfg.Settings, "model", "")
		ollamaClient := l.platformComp.OllamaPlatform(model)
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

func (l *Loader) createTarget(name string, cfg TargetConfig, store *storage.Store) (core.Target, error) {
	switch cfg.Type {
	case "discord":
		if cfg.Platform == "" {
			return nil, fmt.Errorf("platform is required for discord target")
		}

		discordPlatform := l.platformComp.Discord()
		if discordPlatform == nil {
			return nil, fmt.Errorf("discord platform not initialized")
		}

		channelID := GetString(cfg.Settings, "channel_id", "")
		if channelID == "" {
			return nil, fmt.Errorf("channel_id is required for discord target")
		}
		channelType := GetString(cfg.Settings, "channel_type", "text")

		return platforms.NewDiscordTarget(name, platforms.DiscordConfig{
			Platform:    discordPlatform,
			ChannelID:   channelID,
			ChannelType: channelType,
		}), nil

	case "feed":
		return targets.NewFeedTarget(name, store.Feed()), nil

	default:
		return nil, fmt.Errorf("unsupported target type: %s", cfg.Type)
	}
}

func (l *Loader) Shutdown(ctx context.Context) error {
	if l.serverComp != nil {
		if err := l.serverComp.Close(ctx); err != nil {
			log.Printf("[Loader] Error closing server component: %v", err)
		}
	}

	if l.platformComp != nil {
		if err := l.platformComp.Close(ctx); err != nil {
			log.Printf("[Loader] Error closing platform component: %v", err)
		}
	}

	if l.storageComp != nil {
		if err := l.storageComp.Close(ctx); err != nil {
			log.Printf("[Loader] Error closing storage component: %v", err)
		}
	}

	return nil
}

func LoadAndBuild(ctx context.Context, configPath string) (*core.Bot, error) {
	cfg, err := Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	loader := NewLoader(cfg)
	return loader.Initialize(ctx)
}

func (l *Loader) GetStorageComponent() *components.StorageComponent {
	return l.storageComp
}

func (l *Loader) GetPlatformComponent() *components.PlatformComponent {
	return l.platformComp
}

func (l *Loader) GetServerComponent() *components.ServerComponent {
	return l.serverComp
}
