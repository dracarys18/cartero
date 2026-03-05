package types

import (
	"cartero/internal/components"
	"cartero/internal/config"
	"cartero/internal/storage"
	"context"
	"log/slog"
	"sync"
	"time"
)

type Item struct {
	ID              string
	Title           string
	URL             string
	Content         interface{}
	Metadata        map[string]interface{}
	Source          string
	TextContent     *Article
	MatchedKeywords string
	Timestamp       time.Time
	mu              sync.RWMutex
}

type Article struct {
	Text        string
	Image       string
	Description string
}

func (i *Item) GetID() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.ID
}

func (i *Item) GetSource() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.Source
}

func (i *Item) GetContent() interface{} {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.Content
}

func (i *Item) GetTimestamp() time.Time {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.Timestamp
}

func (i *Item) GetArticle() *Article {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.TextContent
}

func (i *Item) GetThumbnail() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.TextContent != nil {
		if i.TextContent.Image != "" {
			return i.TextContent.Image
		}
	}
	return ""
}

func (i *Item) ModifyContent(content any) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.Content = content
}

func (i *Item) GetMetadata(key string) (interface{}, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if i.Metadata == nil {
		return nil, false
	}
	val, ok := i.Metadata[key]
	return val, ok
}

func (i *Item) AddMetadata(key string, value any) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.Metadata == nil {
		i.Metadata = make(map[string]interface{})
	}
	i.Metadata[key] = value
}

func (i *Item) SetArticle(article *Article) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.TextContent = article
}

func (i *Item) GetTitle() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.Title
}

func (i *Item) SetTitle(title string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Title = title
}

func (i *Item) GetURL() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.URL
}

func (i *Item) SetURL(url string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.URL = url
}

func (i *Item) GetMatchedKeywords() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.MatchedKeywords
}

func (i *Item) SetMatchedKeywords(keywords string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.MatchedKeywords = keywords
}

type PublishResult struct {
	Success   bool
	Target    string
	ItemID    string
	Timestamp time.Time
	Error     error
	Metadata  map[string]interface{}
}

type Source interface {
	Name() string
	Initialize(ctx context.Context) error
	Fetch(ctx context.Context, state StateAccessor) (<-chan *Item, <-chan error)
	Shutdown(ctx context.Context) error
}

type Processor interface {
	Name() string
	Process(ctx context.Context, s StateAccessor, item *Item) error
	DependsOn() []string
}

type ProcessorConfig struct {
	Name    string
	Type    string
	Enabled bool
}

type Target interface {
	Name() string
	Initialize(ctx context.Context) error
	Publish(ctx context.Context, item *Item) (*PublishResult, error)
	Shutdown(ctx context.Context) error
}

type Storage interface {
	Store(ctx context.Context, item *Item) error
	Get(ctx context.Context, id string) (*Item, error)
	Exists(ctx context.Context, id string) (bool, error)
	MarkPublished(ctx context.Context, itemID, target string) error
	IsPublished(ctx context.Context, itemID, target string) (bool, error)
	Close() error
}

type StateAccessor interface {
	GetConfig() *config.Config
	GetStorage() storage.StorageInterface
	GetRegistry() *components.Registry
	GetLogger() *slog.Logger
	GetPipeline() interface{}
	GetChain() ProcessorChain
}

type ProcessorChain interface {
	Execute(ctx context.Context, state StateAccessor, item *Item) error
	With(name string, processor Processor) ProcessorChain
	WithMultiple(procs map[string]Processor) ProcessorChain
	Build()
}
