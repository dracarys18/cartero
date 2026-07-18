package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"cartero/internal/storage"
)

const searchLimit = 40

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	key := "search:" + strings.ToLower(query)
	if e, ok := h.cache.get(key); ok {
		writeHTML(w, r, e)
		return
	}

	var entries []storage.FeedEntry
	if query != "" && h.embedder != nil {
		vecs, err := h.embedder.Embed(r.Context(), []string{query})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, "Error: %v", err)
			return
		}
		if len(vecs) > 0 {
			entries, err = h.entryStore.SearchSemantic(r.Context(), vecs[0], searchLimit)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprintf(w, "Error: %v", err)
				return
			}
		}
	}

	base := baseURL(r)
	data := map[string]interface{}{
		"Title":       fmt.Sprintf("cartero — search — %s", query),
		"Query":       query,
		"Entries":     entries,
		"Now":         time.Now(),
		"Page":        1,
		"TotalPages":  1,
		"HasNext":     false,
		"HasPrev":     false,
		"Total":       len(entries),
		"BaseURL":     base,
		"Canonical":   base + "/search",
		"Description": siteDescription,
		"NoIndex":     true,
	}

	html, err := h.renderBytes(data)
	if err != nil {
		fmt.Printf("Template error: %v\n", err)
		return
	}
	writeHTML(w, r, h.cache.set(key, html))
}
