package handler

import (
	"fmt"
	"net/http"
	"time"
)

func (h *Handler) Homepage(w http.ResponseWriter, r *http.Request) {
	h.feedPage(w, r, false)
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	h.feedPage(w, r, true)
}

func (h *Handler) feedPage(w http.ResponseWriter, r *http.Request, archive bool) {
	loc := h.location(r)
	now := time.Now().In(loc)
	q := parseFeedQuery(r, archive, now)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("Vary", "Cookie")

	key := q.cacheKey() + "|" + loc.String()
	if e, ok := h.cache.get(key); ok {
		writeHTML(w, r, e)
		return
	}

	filter := q.storageFilter(now)
	result, err := h.entryStore.ListEntriesPaginated(r.Context(), q.Page, 80, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "Error: %v", err)
		return
	}

	topics, sources, err := h.entryStore.ListFacets(r.Context(), filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "Error: %v", err)
		return
	}

	data := map[string]interface{}{
		"Title":       h.config.SiteName,
		"Query":       "",
		"Entries":     result.Entries,
		"Now":         now,
		"Page":        result.Page,
		"TotalPages":  result.TotalPages,
		"HasNext":     result.HasNext,
		"HasPrev":     result.HasPrevious,
		"PrevURL":     q.pageHref(result.Page - 1),
		"NextURL":     q.pageHref(result.Page + 1),
		"Total":       result.Total,
		"Archive":     q.Archive,
		"TZ":          loc.String(),
		"ArchiveHref": q.archiveToggleHref(),
		"From":        q.From.Format(dateLayout),
		"To":          q.To.Format(dateLayout),
		"MaxDate":     now.Format(dateLayout),
		"RangeLabel":  q.rangeLabel(),
		"DateHidden":  q.dateHidden(),
		"Menus":       []filterMenu{q.topicMenu(topics), q.sourceMenu(sources)},
		"Pills":       q.activePills(),
		"Filtered":    q.filtered(),
		"ClearHref":   q.clearHref(),
		"BaseURL":     h.config.SiteURL,
		"Canonical":   h.config.SiteURL + r.URL.RequestURI(),
		"Description": h.config.SiteDescription,
		"NoIndex":     q.filtered() || q.Archive,
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
