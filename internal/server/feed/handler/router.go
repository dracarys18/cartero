package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/", h.Homepage)
	r.Get("/search", h.Search)
	r.Get("/feed.rss", h.RSSFeed)
	r.Get("/feed.atom", h.AtomFeed)
	r.Get("/feed.json", h.JSONFeed)
	r.Get("/feed.health", h.Health)
	r.Get("/sw.js", h.ServiceWorker)
	r.Get("/robots.txt", h.Robots)
	r.Get("/sitemap.xml", h.Sitemap)

	fileServer := http.FileServer(http.Dir("assets"))
	r.Handle("/assets/*", http.StripPrefix("/assets/", fileServer))

	return r
}
