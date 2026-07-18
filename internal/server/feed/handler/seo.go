package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const siteDescription = "cartero — a hand-tuned feed of the best engineering, systems, and research writing on the web."

const sitemapMaxPages = 50

func baseURL(r *http.Request) string {
	scheme := "https"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + r.Host
}

func (h *Handler) Robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /search\n\nSitemap: %s/sitemap.xml\n", baseURL(r))
}

func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	base := baseURL(r)

	start := time.Unix(0, 0)
	end := time.Now().Add(24 * time.Hour)
	result, err := h.entryStore.ListEntriesPaginated(r.Context(), 1, 80, start, end)

	totalPages := 1
	if err == nil && result != nil && result.TotalPages > totalPages {
		totalPages = result.TotalPages
	}
	if totalPages > sitemapMaxPages {
		totalPages = sitemapMaxPages
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	b.WriteString("  <url><loc>" + base + "/</loc></url>\n")
	for p := 2; p <= totalPages; p++ {
		_, _ = fmt.Fprintf(&b, "  <url><loc>%s/?page=%d</loc></url>\n", base, p)
	}
	b.WriteString("</urlset>\n")

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}
