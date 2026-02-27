package sources

import (
	"cartero/internal/config"
	"cartero/internal/lua"
	"cartero/internal/types"
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/cjoudrey/gluahttp"
	json "layeh.com/gopher-json"
)

type ScraperSource struct {
	name        string
	scraperType string
	scraperName string
	scriptPath  string
	config      map[string]interface{}
	maxItems    int
	runtime     *lua.Runtime
	logger      *slog.Logger
	embeddedFS  embed.FS
}

func NewScraperSource(name string, settings config.SourceSettings, embeddedFS embed.FS, logger *slog.Logger) (*ScraperSource, error) {
	s := &ScraperSource{
		name:        name,
		scraperType: settings.ScraperType,
		scraperName: settings.ScraperName,
		scriptPath:  settings.ScriptPath,
		config:      settings.Config,
		maxItems:    settings.MaxItems,
		embeddedFS:  embeddedFS,
		logger:      logger,
	}

	if err := s.validate(); err != nil {
		return nil, err
	}

	if s.config == nil {
		s.config = make(map[string]interface{})
	}

	if s.maxItems > 0 {
		s.config["max_items"] = s.maxItems
	}

	return s, nil
}

func (s *ScraperSource) validate() error {
	if s.scraperType == "" {
		return fmt.Errorf("scraper_type is required")
	}

	switch s.scraperType {
	case "internal":
		if s.scraperName == "" {
			return fmt.Errorf("scraper_name is required when scraper_type is internal")
		}
		if s.scriptPath != "" {
			return fmt.Errorf("script_path must not be set when scraper_type is internal")
		}
	case "external":
		if s.scriptPath == "" {
			return fmt.Errorf("script_path is required when scraper_type is external")
		}
		if s.scraperName != "" {
			return fmt.Errorf("scraper_name must not be set when scraper_type is external")
		}
	default:
		return fmt.Errorf("invalid scraper_type: %s (must be 'internal' or 'external')", s.scraperType)
	}

	return nil
}

func (s *ScraperSource) Name() string {
	return s.name
}

func (s *ScraperSource) Initialize(ctx context.Context) error {
	var loader lua.Loader

	switch s.scraperType {
	case "internal":
		loader = lua.NewEmbeddedLoader(s.embeddedFS, "scripts/scrapers")
	case "external":
		loader = lua.NewFilesystemLoader(".")
	default:
		return fmt.Errorf("invalid scraper_type: %s", s.scraperType)
	}

	s.runtime = lua.NewRuntime(
		lua.WithLoader(loader),
		lua.WithSecureMode(true),
	)

	L := s.runtime.State()

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	L.PreloadModule("http", gluahttp.NewHttpModule(httpClient).Loader)

	json.Preload(L)

	if err := lua.RegisterModule(L, lua.NewHTMLModule()); err != nil {
		return fmt.Errorf("failed to register HTML module: %w", err)
	}

	if err := lua.RegisterModule(L, lua.NewLogModule(s.logger)); err != nil {
		return fmt.Errorf("failed to register Log module: %w", err)
	}

	scriptContent, err := s.loadScript(loader)
	if err != nil {
		return err
	}

	if err := s.runtime.LoadScript(scriptContent); err != nil {
		return fmt.Errorf("failed to load script: %w", err)
	}

	return nil
}

func (s *ScraperSource) loadScript(loader lua.Loader) (string, error) {
	var identifier string

	switch s.scraperType {
	case "internal":
		identifier = s.scraperName
	case "external":
		identifier = s.scriptPath
	default:
		return "", fmt.Errorf("invalid scraper_type: %s", s.scraperType)
	}

	scriptContent, err := loader.Load(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to load script: %w", err)
	}

	return scriptContent, nil
}

func (s *ScraperSource) Fetch(ctx context.Context, state types.StateAccessor) (<-chan *types.Item, <-chan error) {
	itemChan := make(chan *types.Item)
	errChan := make(chan error, 1)

	go func() {
		defer close(itemChan)
		defer close(errChan)

		results, err := s.runtime.Execute("scrape", s.config)
		if err != nil {
			s.logger.Error("Scraper execution error", "source", s.name, "error", err)
			errChan <- err
			return
		}

		if len(results) == 0 {
			s.logger.Warn("Scraper returned no results", "source", s.name)
			return
		}

		s.logger.Debug("Scraper returned results", "source", s.name, "count", len(results), "type", fmt.Sprintf("%T", results[0]))

		items, err := s.convertResultsToItems(results[0])
		if err != nil {
			s.logger.Error("Failed to convert scraper results", "source", s.name, "error", err)
			errChan <- err
			return
		}

		s.logger.Debug("Scraper fetched items", "source", s.name, "count", len(items))

		for _, item := range items {
			select {
			case itemChan <- item:
				s.logger.Debug("Scraper sent item", "source", s.name, "id", item.ID)
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return itemChan, errChan
}

func (s *ScraperSource) convertResultsToItems(result interface{}) ([]*types.Item, error) {
	resultSlice, ok := result.([]interface{})
	if !ok {
		resultMap, ok := result.(map[string]interface{})
		if ok && len(resultMap) == 0 {
			s.logger.Debug("Scraper returned empty table", "source", s.name)
			return []*types.Item{}, nil
		}
		return nil, fmt.Errorf("expected array of items, got %T", result)
	}

	items := make([]*types.Item, 0, len(resultSlice))

	for i, itemData := range resultSlice {
		itemMap, ok := itemData.(map[string]interface{})
		if !ok {
			s.logger.Warn("Skipping invalid item", "source", s.name, "index", i, "type", fmt.Sprintf("%T", itemData))
			continue
		}

		item := s.convertMapToItem(itemMap)
		if item != nil {
			items = append(items, item)
		}
	}

	return items, nil
}

func (s *ScraperSource) convertMapToItem(itemMap map[string]interface{}) *types.Item {
	id, _ := itemMap["id"].(string)
	title, _ := itemMap["title"].(string)
	url, _ := itemMap["url"].(string)

	if id == "" || title == "" {
		s.logger.Warn("Skipping item with missing required fields", "source", s.name, "id", id, "title", title)
		return nil
	}

	item := &types.Item{
		ID:        id,
		Source:    s.name,
		Timestamp: time.Now(),
		Content:   itemMap,
		Metadata:  make(map[string]interface{}),
	}

	if title != "" {
		item.Metadata["title"] = title
	}
	if url != "" {
		item.Metadata["url"] = url
	}

	if author, ok := itemMap["author"].(string); ok && author != "" {
		item.Metadata["author"] = author
	}

	if published, ok := itemMap["published"].(string); ok && published != "" {
		item.Metadata["published"] = published
	}

	if thumbnail, ok := itemMap["thumbnail"].(string); ok && thumbnail != "" {
		if item.TextContent == nil {
			item.TextContent = &types.Article{}
		}
		item.TextContent.Image = thumbnail
	}

	if content, ok := itemMap["content"].(string); ok && content != "" {
		if item.TextContent == nil {
			item.TextContent = &types.Article{}
		}
		item.TextContent.Text = content
	}

	if metadata, ok := itemMap["metadata"].(map[string]interface{}); ok {
		for key, value := range metadata {
			item.Metadata[key] = value
		}
	}

	return item
}

func (s *ScraperSource) Shutdown(ctx context.Context) error {
	if s.runtime != nil {
		return s.runtime.Close()
	}
	return nil
}
