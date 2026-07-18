package handler

import (
	"time"

	"cartero/internal/platforms"
	"cartero/internal/storage"
	"cartero/internal/template"
)

const renderCacheTTL = 60 * time.Second

type Config struct {
	Name     string
	FeedSize int
	MaxItems int
}

type Handler struct {
	config     Config
	entryStore storage.EntryStore
	embedder   platforms.Embedder
	tmpl       *template.Template
	cache      *pageCache
}

func New(config Config, entryStore storage.EntryStore, embedder platforms.Embedder) *Handler {
	tmpl := &template.Template{}
	if err := tmpl.Load("templates/homepage.gotmpl", template.HtmlTemplate, funcMap()); err != nil {
		panic(err.Error())
	}

	return &Handler{
		config:     config,
		entryStore: entryStore,
		embedder:   embedder,
		tmpl:       tmpl,
		cache:      newPageCache(renderCacheTTL),
	}
}
