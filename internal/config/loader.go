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
	config   *Config
	registry *components.Registry
	pipeline *core.Pipeline
}

func NewLoader(cfg *Config) *Loader {
	return &Loader{
		config:   cfg,
		registry: components.NewRegistry(),
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

	storageComp := components.NewStorageComponent(l.config.Storage.Path)
	if err := l.registry.Register(storageComp); err != nil {
		return fmt.Errorf("failed to register storage component: %w", err)
	}

	platformComp := l.buildPlatformComponent()
	if err := l.registry.Register(platformComp); err != nil {
		return fmt.Errorf("failed to register platform component: %w", err)
	}

	serverComp := components.NewServerComponent(l.registry)
	for name, targetCfg := range l.config.Targets {
		if targetCfg.Type != "feed" {
			continue
		}

		port := GetString(targetCfg.Settings, "port", "8080")
		feedSize := GetInt(targetCfg.Settings, "feed_size", 100)
		maxItems := GetInt(targetCfg.Settings, "max_items", 50)

		serverComp.Register(components.ServerConfig{
			Name:     name,
			Port:     port,
			FeedSize: feedSize,
			MaxItems: maxItems,
		})
	}

	if err := l.registry.Register(serverComp); err != nil {
		return fmt.Errorf("failed to register server component: %w", err)
	}

	if err := l.registry.InitializeAll(ctx); err != nil {
		return fmt.Errorf("component initialization failed: %w", err)
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
	store := l.registry.Get(components.StorageComponentName).(*components.StorageComponent).Store()
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
	store := l.registry.Get(components.StorageComponentName).(*components.StorageComponent).Store()

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
		platformComponent := l.registry.Get(components.PlatformComponentName).(*components.PlatformComponent)
		model := GetString(cfg.Settings, "model", "")
		ollamaClient := platformComponent.OllamaPlatform(model)
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
		return processors.NewContentDedupeProcessor(name, GetString(cfg.Settings, "field", "")), nil

	case "rate_limit":
		return processors.NewRateLimitProcessor(name, GetInt(cfg.Settings, "limit", 10), GetDuration(cfg.Settings, "window", 1*time.Minute)), nil

	case "token_bucket":
		return processors.NewTokenBucketProcessor(name, GetInt(cfg.Settings, "capacity", 10), GetDuration(cfg.Settings, "refill_rate", 1*time.Second)), nil

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

		platformComponent := l.registry.Get(components.PlatformComponentName).(*components.PlatformComponent)
		discordPlatform := platformComponent.Discord()
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
	return l.registry.CloseAll(ctx)
}

func LoadAndBuild(ctx context.Context, configPath string) (*core.Bot, error) {
	cfg, err := Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	loader := NewLoader(cfg)
	return loader.Initialize(ctx)
}

func (l *Loader) GetComponent(name string) components.IComponent {
	return l.registry.Get(name)
}
