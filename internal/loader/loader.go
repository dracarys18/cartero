package loader

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cartero/internal/components"
	"cartero/internal/config"
	"cartero/internal/core"
	"cartero/internal/processors"
	"cartero/internal/sources"
	"cartero/internal/state"
	"cartero/internal/targets"
)

type Loader struct {
	config *config.Config
}

func NewLoader(cfg *config.Config) *Loader {
	return &Loader{
		config: cfg,
	}
}

func (l *Loader) Initialize(ctx context.Context) (*state.State, error) {
	registry := components.NewRegistry()
	slog.Info("Initializing all components")

	storageComp := components.NewStorageComponent(l.config.Storage.Path)
	if err := registry.Register(storageComp); err != nil {
		return nil, fmt.Errorf("failed to register storage component: %w", err)
	}

	platformComp := l.buildPlatformComponent()
	if err := registry.Register(platformComp); err != nil {
		return nil, fmt.Errorf("failed to register platform component: %w", err)
	}

	serverComp := components.NewServerComponent(registry)
	for name, targetCfg := range l.config.Targets {
		if targetCfg.Type != "feed" {
			continue
		}

		cfg := targetCfg.Settings.FeedTargetSettings

		serverComp.Register(components.ServerConfig{
			Name:     name,
			Port:     cfg.Port,
			FeedSize: cfg.FeedSize,
			MaxItems: cfg.MaxItems,
		})
	}

	if err := registry.Register(serverComp); err != nil {
		return nil, fmt.Errorf("failed to register server component: %w", err)
	}

	if err := registry.InitializeAll(ctx); err != nil {
		return nil, fmt.Errorf("component initialization failed: %w", err)
	}

	slog.Info("All components initialized successfully")

	pipeline, err := l.buildPipeline(ctx, registry)
	appState := state.NewState(l.config, registry, pipeline)

	if err != nil {
		return nil, fmt.Errorf("failed to build pipeline: %w", err)
	}

	appState.Pipeline = pipeline

	return appState, nil
}

func (l *Loader) buildPlatformComponent() *components.PlatformComponent {
	return components.NewPlatformComponent(l.config.Platforms)
}

func (l *Loader) buildPipeline(_ context.Context, registry *components.Registry) (*core.Pipeline, error) {
	store := registry.Get(components.StorageComponentName).(*components.StorageComponent).Store()
	pipeline := core.NewPipeline(store.Items())

	if err := l.addProcessors(pipeline, registry); err != nil {
		return nil, fmt.Errorf("failed to add processors: %w", err)
	}

	if err := l.addSourceRoutes(pipeline, registry); err != nil {
		return nil, fmt.Errorf("failed to add source routes: %w", err)
	}

	return pipeline, nil
}

func (l *Loader) addProcessors(pipeline *core.Pipeline, registry *components.Registry) error {
	for name, processorCfg := range l.config.Processors {
		if !processorCfg.Enabled {
			continue
		}

		processor, err := l.createProcessor(name, processorCfg, registry)
		if err != nil {
			return fmt.Errorf("failed to create processor %s: %w", name, err)
		}

		pipelineConfig := core.ProcessorConfig{
			Name:    name,
			Type:    processorCfg.Type,
			Enabled: processorCfg.Enabled,
		}
		pipeline.AddProcessorWithConfig(processor, pipelineConfig)
	}

	return nil
}

func (l *Loader) addSourceRoutes(pipeline *core.Pipeline, registry *components.Registry) error {
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

			target, err := l.createTarget(targetName, targetCfg, registry)
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

func (l *Loader) createSource(name string, cfg config.SourceConfig) (core.Source, error) {
	maxItems := cfg.Settings.MaxItems

	switch cfg.Type {
	case "hackernews":
		s := cfg.Settings.HackerNewsSettings
		storyType := s.StoryType
		if storyType == "" {
			storyType = "topstories"
		}
		return sources.NewHackerNewsSource(name, storyType, maxItems), nil

	case "lobsters":
		s := cfg.Settings.LobstersSettings
		return sources.NewLobstersSource(name, maxItems, s.SortBy, s.IncludeCategories, s.ExcludeCategories), nil

	case "lesswrong":
		return sources.NewLessWrongSource(name, maxItems), nil

	case "rss":
		s := cfg.Settings.RSSSettings
		if s.FeedURL == "" {
			return nil, fmt.Errorf("feed_url is required for RSS source")
		}
		return sources.NewRSSSource(name, s.FeedURL, maxItems), nil

	default:
		return nil, fmt.Errorf("unsupported source type: %s", cfg.Type)
	}
}

func (l *Loader) createProcessor(name string, cfg config.ProcessorConfig, registry *components.Registry) (core.Processor, error) {
	switch cfg.Type {
	case "summary":
		s := cfg.Settings.SummarySettings
		return processors.NewSummaryProcessor(name, s.Model, registry), nil

	case "extract_fields":
		s := cfg.Settings.ExtractFieldsSettings
		return processors.FieldExtractor(name, s.Fields), nil

	case "template":
		s := cfg.Settings.TemplateSettings
		if s.Template == "" {
			return nil, fmt.Errorf("template is required for template processor")
		}
		return processors.TemplateTransformer(name, s.Template), nil

	case "enrich":
		return nil, fmt.Errorf("enrich processor not supported in strict config mode yet")

	case "filter_score":
		s := cfg.Settings.ScoreFilterSettings
		return processors.NewScoreFilterProcessor(name, s.MinScore), nil

	case "filter_keyword":
		s := cfg.Settings.KeywordFilterSettings
		return processors.NewKeywordFilterProcessor(name, s), nil

	case "dedupe":
		s := cfg.Settings.DedupeSettings
		ttl := config.ParseDuration(s.TTL, 24*time.Hour)
		return processors.NewDedupeProcessor(name, ttl), nil

	case "rate_limit":
		s := cfg.Settings.RateLimitSettings
		window := config.ParseDuration(s.Window, 1*time.Minute)
		return processors.NewRateLimitProcessor(name, s.Limit, window), nil

	case "token_bucket":
		s := cfg.Settings.TokenBucketSettings
		rate := config.ParseDuration(s.RefillRate, 1*time.Second)
		return processors.NewTokenBucketProcessor(name, s.Capacity, rate), nil

	case "extract_text":
		limit := cfg.Settings.ExtractTextSettings.Limit
		return processors.NewExtractProcessor(name, limit), nil
	default:
		return nil, fmt.Errorf("unsupported processor type: %s", cfg.Type)
	}
}

func (l *Loader) createTarget(name string, cfg config.TargetConfig, re *components.Registry) (core.Target, error) {
	switch cfg.Type {
	case "discord":
		s := cfg.Settings.DiscordTargetSettings

		if s.ChannelID == "" {
			return nil, fmt.Errorf("channel_id is required for discord target")
		}

		return targets.NewDiscordTarget(name, s.ChannelID, s.ChannelType, re), nil

	case "feed":
		return targets.NewFeedTarget(name, re), nil

	default:
		return nil, fmt.Errorf("unsupported target type: %s", cfg.Type)
	}
}

func LoadAndBuild(ctx context.Context, configPath string) (*state.State, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	loader := NewLoader(cfg)
	return loader.Initialize(ctx)
}
