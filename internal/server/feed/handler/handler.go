package handler

import (
	"time"

	"cartero/internal/storage"
	"cartero/internal/template"
)

const renderCacheTTL = 60 * time.Second

type Config struct {
	Name            string
	FeedSize        int
	MaxItems        int
	SiteURL         string
	SiteName        string
	SiteDescription string
}

type Handler struct {
	config     Config
	entryStore storage.EntryStore
	tmpl       *template.Template
	readerTmpl *template.Template
	cache      *pageCache
}

func New(config Config, entryStore storage.EntryStore) *Handler {
	tmpl := &template.Template{}
	if err := tmpl.Load("templates/homepage.gotmpl", template.HtmlTemplate, funcMap()); err != nil {
		panic(err.Error())
	}

	readerTmpl := &template.Template{}
	if err := readerTmpl.Load("templates/reader.gotmpl", template.HtmlTemplate, funcMap()); err != nil {
		panic(err.Error())
	}

	return &Handler{
		config:     config,
		entryStore: entryStore,
		tmpl:       tmpl,
		readerTmpl: readerTmpl,
		cache:      newPageCache(renderCacheTTL),
	}
}
