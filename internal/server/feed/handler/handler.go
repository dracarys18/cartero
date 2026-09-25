package handler

import (
	"net/http"
	"sync"
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
	locations  sync.Map
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

const tzCookie = "tz"

func (h *Handler) location(r *http.Request) *time.Location {
	c, err := r.Cookie(tzCookie)
	if err != nil {
		return time.UTC
	}
	if loc, ok := h.locations.Load(c.Value); ok {
		return loc.(*time.Location)
	}
	loc, err := time.LoadLocation(c.Value)
	if err != nil {
		return time.UTC
	}
	h.locations.Store(c.Value, loc)
	return loc
}
