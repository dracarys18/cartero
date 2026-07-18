package handler

import (
	"fmt"
	"net/http"
	"time"
)

func (h *Handler) Homepage(w http.ResponseWriter, r *http.Request) {
	page := parsePageParam(r)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")

	key := fmt.Sprintf("home:%d", page)
	if e, ok := h.cache.get(key); ok {
		writeHTML(w, r, e)
		return
	}

	start := time.Unix(0, 0)
	end := time.Now().Add(24 * time.Hour)

	result, err := h.entryStore.ListEntriesPaginated(r.Context(), page, 80, start, end)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "Error: %v", err)
		return
	}

	data := map[string]interface{}{
		"Title":       h.config.SiteName,
		"Query":       "",
		"Entries":     result.Entries,
		"Now":         time.Now(),
		"Page":        result.Page,
		"TotalPages":  result.TotalPages,
		"HasNext":     result.HasNext,
		"HasPrev":     result.HasPrevious,
		"Total":       result.Total,
		"BaseURL":     h.config.SiteURL,
		"Canonical":   h.config.SiteURL + r.URL.RequestURI(),
		"Description": h.config.SiteDescription,
		"NoIndex":     false,
	}

	html, err := h.renderBytes(data)
	if err != nil {
		fmt.Printf("Template error: %v\n", err)
		return
	}
	writeHTML(w, r, h.cache.set(key, html))
}

func parsePageParam(r *http.Request) int {
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		return 1
	}
	page := 1
	_, _ = fmt.Sscanf(pageStr, "%d", &page)
	if page < 1 {
		page = 1
	}
	return page
}
