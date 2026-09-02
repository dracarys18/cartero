package handler

import (
	"net/http"
)

// ReaderShell serves the reader page shell. The article is fetched and
// rendered entirely in the browser (assets/reader.js + Readability.js), so the
// server only supplies the page structure and static assets.
func (h *Handler) ReaderShell(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")

	data := map[string]interface{}{
		"Title":       "Read · " + h.config.SiteName,
		"SiteName":    h.config.SiteName,
		"Description": h.config.SiteDescription,
		"BaseURL":     h.config.SiteURL,
		"Canonical":   h.config.SiteURL + r.URL.RequestURI(),
		"NoIndex":     true,
	}

	if h.readerTmpl == nil {
		http.Error(w, "reader template not loaded", http.StatusInternalServerError)
		return
	}
	if err := h.readerTmpl.HTMLTemplate().Execute(w, data); err != nil {
		http.Error(w, "failed to render reader", http.StatusInternalServerError)
	}
}
