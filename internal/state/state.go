package state

import (
	"context"
	"fmt"

	"cartero/internal/components"
	"cartero/internal/config"
	"cartero/internal/core"
	"cartero/internal/middleware"
	"cartero/internal/processors"
	"cartero/internal/sources"
	"cartero/internal/storage"
	_ "cartero/internal/storage/sqlite"
	"cartero/internal/targets"
	"cartero/internal/types"
	"log/slog"
)

type State struct {
	Config   *config.Config
	Registry *components.Registry
	Pipeline *core.Pipeline
	Storage  storage.StorageInterface
	Chain    types.ProcessorChain
	Logger   *slog.Logger
}

func New(logger *slog.Logger) *State {
	return &State{
		Logger: logger,
	}
}

func (s *State) Initialize(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	s.Config = cfg

	st, err := storage.New(ctx, s.Config.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	s.Storage = st

	s.Registry = components.NewRegistry()

	storageComp := components.NewStorageComponent(s.Storage)
	if err := s.Registry.Register(storageComp); err != nil {
		return fmt.Errorf("failed to register storage component: %w", err)
	}

	platformComp := s.buildPlatformComponent()
	if err := s.Registry.Register(platformComp); err != nil {
		return fmt.Errorf("failed to register platform component: %w", err)
	}

	serverComp := components.NewServerComponent(s.Registry)
	for name, targetCfg := range s.Config.Targets {
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

	if err := s.Registry.Register(serverComp); err != nil {
		return fmt.Errorf("failed to register server component: %w", err)
	}

	if err := s.Registry.InitializeAll(ctx); err != nil {
		return fmt.Errorf("component initialization failed: %w", err)
	}

	pipeline, err := s.buildPipeline(ctx)
	if err != nil {
		return fmt.Errorf("failed to build pipeline: %w", err)
	}
	s.Pipeline = pipeline

	chain := s.buildProcessorChain(ctx)
	s.Chain = chain

	return nil
}

func (s *State) GetConfig() *config.Config {
	return s.Config
}

func (s *State) GetStorage() storage.StorageInterface {
	return s.Storage
}

func (s *State) GetRegistry() *components.Registry {
	return s.Registry
}

func (s *State) GetPipeline() interface{} {
	return s.Pipeline
}

func (s *State) GetChain() types.ProcessorChain {
	return s.Chain
}

func (s *State) GetLogger() *slog.Logger {
	return s.Logger
}

func (s *State) buildPlatformComponent() *components.PlatformComponent {
	return components.NewPlatformComponent(s.Config.Platforms)
}

func (s *State) buildPipeline(ctx context.Context) (*core.Pipeline, error) {
	pipeline := core.NewPipeline()

	for sourceName, sourceCfg := range s.Config.Sources {
		if !sourceCfg.Enabled {
			continue
		}

		source := s.createSource(sourceName, sourceCfg)
		if source == nil {
			return nil, fmt.Errorf("failed to create source %s", sourceName)
		}

		var routeTargets []types.Target
		if len(sourceCfg.Targets) == 0 {
			return nil, fmt.Errorf("source %s has no targets configured", sourceName)
		}

		for _, targetName := range sourceCfg.Targets {
			targetCfg, exists := s.Config.Targets[targetName]
			if !exists {
				return nil, fmt.Errorf("target %s not found in config for source %s", targetName, sourceName)
			}
			if !targetCfg.Enabled {
				continue
			}

			target := s.createTarget(targetName, targetCfg)
			if target == nil {
				return nil, fmt.Errorf("failed to create target %s for source %s", targetName, sourceName)
			}

			routeTargets = append(routeTargets, target)
		}

		if len(routeTargets) == 0 {
			return nil, fmt.Errorf("source %s has no enabled targets", sourceName)
		}

		route := core.SourceRoute{
			Source:  source,
			Targets: routeTargets,
		}
		pipeline.AddRoute(route)
	}

	return pipeline, nil
}

func (s *State) buildProcessorChain(ctx context.Context) types.ProcessorChain {
	chain := middleware.New(s)

	for name, procCfg := range s.Config.Processors {
		if !procCfg.Enabled {
			continue
		}

		processor := s.createProcessor(name, procCfg)
		if processor != nil {
			chain = chain.With(procCfg.Type, processor)
		}
	}

	chain.Build()

	return chain
}

func (s *State) createSource(name string, cfg config.SourceConfig) types.Source {
	maxItems := cfg.Settings.MaxItems

	switch cfg.Type {
	case "hackernews":
		storyType := cfg.Settings.StoryType
		if storyType == "" {
			storyType = "topstories"
		}
		return sources.NewHackerNewsSource(name, storyType, maxItems)

	case "lobsters":
		lobCfg := cfg.Settings.LobstersSettings
		return sources.NewLobstersSource(name, maxItems, lobCfg.SortBy, lobCfg.IncludeCategories, lobCfg.ExcludeCategories)

	case "lesswrong":
		return sources.NewLessWrongSource(name, maxItems)

	case "rss":
		rssCfg := cfg.Settings.RSSSettings

		if rssCfg.From.Type != "" {
			source, err := sources.NewRSSSourceFromConfig(name, rssCfg, maxItems)
			if err != nil {
				s.Logger.Error("Failed to create RSS source from config", "source", name, "error", err)
				return nil
			}
			return source
		}

		if rssCfg.FeedURL != "" {
			return sources.NewRSSSource(name, rssCfg.FeedURL, maxItems)
		}

		return nil

	default:
		return nil
	}
}

func (s *State) createProcessor(name string, cfg config.ProcessorConfig) types.Processor {
	switch cfg.Type {
	case "summary":
		return processors.NewSummaryProcessor(name)

	case "extract_fields":
		fieldsCfg := cfg.Settings.ExtractFieldsSettings
		return processors.FieldExtractor(name, fieldsCfg.Fields)

	case "template":
		templateCfg := cfg.Settings.TemplateSettings
		if templateCfg.Template == "" {
			return nil
		}
		return processors.TemplateTransformer(name, templateCfg.Template)

	case "filter_score":
		return processors.NewScoreFilterProcessor(name)

	case "filter_keyword":
		return processors.NewKeywordFilterProcessor(name)

	case "filter_published":
		return processors.NewPublishedAtFilterProcessor(name)

	case "dedupe":
		return processors.NewDedupeProcessor(name)

	case "rate_limit":
		return processors.NewRateLimitProcessor(name)

	case "token_bucket":
		return processors.NewTokenBucketProcessor(name)

	case "extract_text":
		return processors.NewExtractProcessor(name)

	default:
		return nil
	}
}

func (s *State) createTarget(name string, cfg config.TargetConfig) types.Target {
	switch cfg.Type {
	case "discord":
		discordCfg := cfg.Settings.DiscordTargetSettings
		if discordCfg.ChannelID == "" {
			return nil
		}
		return targets.NewDiscordTarget(name, discordCfg.ChannelID, discordCfg.ChannelType, s.Registry)

	case "feed":
		return targets.NewFeedTarget(name, s.Registry)

	case "bluesky":
		bskyCfg := cfg.Settings.BlueskyTargetSettings
		return targets.NewBlueskyTarget(name, bskyCfg.Languages, s.Registry)

	default:
		return nil
	}
}
