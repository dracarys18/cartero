package state

import (
	"context"
	"embed"
	"fmt"

	"cartero/internal/components"
	"cartero/internal/config"
	"cartero/internal/core"
	"cartero/internal/processors"
	"cartero/internal/processors/filters"
	"cartero/internal/processors/names"
	"cartero/internal/queue"
	"cartero/internal/sources"
	"cartero/internal/storage"
	_ "cartero/internal/storage/postgres"
	_ "cartero/internal/storage/sqlite"
	"cartero/internal/targets"
	"cartero/internal/types"
	"log/slog"
)

type State struct {
	Config          *config.Config
	Registry        *components.Registry
	Pipeline        *core.Pipeline
	Storage         storage.StorageInterface
	Filters         *filters.Chain
	Queue           *queue.Queue
	RedisConn       *queue.RedisConnection
	Blocklist       types.Blocklist
	Logger          *slog.Logger
	EmbeddedScripts embed.FS
}

func New(logger *slog.Logger, embeddedScripts embed.FS) *State {
	return &State{
		Logger:          logger,
		EmbeddedScripts: embeddedScripts,
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

	conn, err := queue.NewRedisConnection(s.Config.Redis.Addr, s.Config.Redis.Password, s.Config.Redis.DB)
	if err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}
	s.RedisConn = conn
	s.Queue = queue.New(conn)

	if len(s.Config.Blocklist.Domains) > 0 {
		bl := queue.NewBlocklist(conn.Client(), s.Queue.Prefix()+":blocklist")
		if err := bl.Load(ctx, s.Config.Blocklist.Domains); err != nil {
			return fmt.Errorf("failed to load blocklist: %w", err)
		}
		s.Blocklist = bl
	}

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

	if err := s.Pipeline.Initialize(ctx, s.Logger); err != nil {
		return fmt.Errorf("failed to initialize pipeline: %w", err)
	}

	s.Filters = s.buildFilterChain(ctx)

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

func (s *State) GetFilterChain() *filters.Chain {
	return s.Filters
}

func (s *State) GetLogger() *slog.Logger {
	return s.Logger
}

func (s *State) GetQueue() types.Queue {
	return s.Queue
}

func (s *State) GetBlocklist() types.Blocklist {
	return s.Blocklist
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

func (s *State) buildFilterChain(ctx context.Context) *filters.Chain {
	var fs []filters.Processor

	for _, procCfg := range s.Config.Processors {
		if !procCfg.Enabled {
			continue
		}

		processor := s.createProcessor(procCfg)
		if processor == nil {
			continue
		}

		fs = append(fs, processor)
	}

	var targetNames []string
	for name, tc := range s.Config.Targets {
		if tc.Enabled {
			targetNames = append(targetNames, name)
		}
	}
	fs = append(fs, filters.NewPublishedDedupeFilter(targetNames))
	fs = append(fs, filters.NewBlocklistFilter())
	fs = append(fs, processors.NewExtractProcessor(s.Config.Processors[names.ExtractText].Settings.ExtractTextSettings))

	pc := s.Registry.Get(components.PlatformComponentName).(*components.PlatformComponent)
	fs = append(fs,
		filters.NewRankFilter(pc.Embedder(), s.Config.Interests),
		filters.NewDiversifyFilter(),
	)

	return filters.NewChain(fs...)
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

	case "scraper":
		source, err := sources.NewScraperSource(name, cfg.Settings, s.EmbeddedScripts, s.Logger)
		if err != nil {
			s.Logger.Error("Failed to create scraper source", "source", name, "error", err)
			return nil
		}
		return source

	default:
		return nil
	}
}

func (s *State) createProcessor(cfg config.ProcessorConfig) filters.Processor {
	switch cfg.Type {
	case names.Summary:
		return processors.NewSummaryProcessor(cfg.Type, cfg.Settings.SummarySettings)

	case names.FieldExtractor:
		return processors.FieldExtractor(cfg.Type, cfg.Settings.Fields)

	case names.TemplateTransformer:
		if cfg.Settings.Template == "" {
			return nil
		}
		return processors.TemplateTransformer(cfg.Type, cfg.Settings.Template)

	case names.ScoreFilter:
		return processors.NewScoreFilterProcessor(cfg.Type, cfg.Settings.ScoreFilterSettings)

	case "filter_published":
		return processors.NewPublishedAtFilterProcessor(cfg.Type, cfg.Settings.PublishedAtFilterSettings)

	case names.Dedupe:
		return processors.NewDedupeProcessor(cfg.Type)

	case names.EmbedDedupe:
		return processors.NewEmbedDedupeProcessor(cfg.Type, cfg.Settings.DedupeSettings)

	case names.EmbedText:
		return processors.NewEmbedTextProcessor(cfg.Type, cfg.Settings.EmbedTextSettings)

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

	case "telegram":
		tgCfg := cfg.Settings.TelegramTargetSettings
		if tgCfg.ChatID == 0 {
			return nil
		}
		return targets.NewTelegramTarget(name, tgCfg.ChatID, s.Registry)

	default:
		return nil
	}
}
