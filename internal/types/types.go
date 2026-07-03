package types

import (
	"cartero/internal/components"
	"cartero/internal/config"
	"cartero/internal/storage"
	strutils "cartero/internal/utils/string"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
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
	Route           string
	TextContent     *Article
	MatchedKeywords string
	Timestamp       time.Time
	Embedding       [][]float32 `json:"-"`
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

	return strutils.Readable(i.Source)
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

func (i *Item) ModifyContent(content any) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.Content = content
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

func (i *Item) GetEmbedding() [][]float32 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.Embedding
}

func (i *Item) metaString(key string) string {
	if i.Metadata == nil {
		return ""
	}
	if v, ok := i.Metadata[key].(string); ok {
		return v
	}
	return ""
}

func (i *Item) GetLink() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.URL != "" {
		return i.URL
	}
	return i.metaString("link")
}

func (i *Item) GetDescription() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.metaString("description")
}

func (i *Item) GetFeedContent() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.TextContent != nil && i.TextContent.Text != "" {
		return i.TextContent.Text
	}
	return i.metaString("summary")
}

func (i *Item) GetAuthor() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.metaString("author")
}

func (i *Item) GetImageURL() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.TextContent != nil {
		return i.TextContent.Image
	}
	return ""
}

func (i *Item) SetEmbedding(v [][]float32) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Embedding = v
}

type PublishResult struct {
	Success  bool
	Error    error
	Metadata map[string]interface{}
}

type Source interface {
	Name() string
	Initialize(ctx context.Context) error
	Publish(ctx context.Context, state StateAccessor) error
	Shutdown(ctx context.Context) error
}

type Processor interface {
	Name() string
	Initialize(ctx context.Context, st StateAccessor) error
	Process(ctx context.Context, s StateAccessor, item *Item) error
	DependsOn() []string
}

type Target interface {
	Name() string
	Initialize(ctx context.Context) error
	Publish(ctx context.Context, item *Item) (*PublishResult, error)
	Shutdown(ctx context.Context) error
}

type Envelope struct {
	Item    *Item    `json:"item"`
	Targets []string `json:"targets"`
}

func (e *Envelope) TryFrom(fields map[string]any) error {
	itemStr, ok := fields["item"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid item field")
	}
	if err := json.Unmarshal([]byte(itemStr), &e.Item); err != nil {
		return fmt.Errorf("failed to unmarshal item: %w", err)
	}
	if t, ok := fields["targets"].(string); ok && t != "" {
		e.Targets = strings.Split(t, ",")
	}
	return nil
}

func (e Envelope) TryInto() (map[string]any, error) {
	item, err := json.Marshal(e.Item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item: %w", err)
	}
	return map[string]any{
		"item":    string(item),
		"targets": strings.Join(e.Targets, ","),
	}, nil
}

type Queue interface {
	SourceStream() string
	ProcessedStream() string
	CreateGroup(ctx context.Context, stream string) error
	Publish(ctx context.Context, stream string, env Envelope) error
	Consume(ctx context.Context, stream string) ([]Envelope, []string, error)
	Ack(ctx context.Context, stream string, ids ...string) error
	Close() error
}

type Blocklist interface {
	Blocked(ctx context.Context, link string) bool
}

type StateAccessor interface {
	GetConfig() *config.Config
	GetStorage() storage.StorageInterface
	GetRegistry() *components.Registry
	GetLogger() *slog.Logger
	GetPipeline() interface{}
	GetChain() ProcessorChain
	GetQueue() Queue
	GetBlocklist() Blocklist
}

type ProcessorChain interface {
	Execute(ctx context.Context, state StateAccessor, item *Item) error
	With(name string, processor Processor) ProcessorChain
	WithMultiple(procs map[string]Processor) ProcessorChain
	Build()
}
