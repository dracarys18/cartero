package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	utils "cartero/internal/utils/string"
)

const searchLimit = 40

type searchResult struct {
	Title           string `json:"title"`
	Link            string `json:"link"`
	Source          string `json:"source"`
	Age             string `json:"age"`
	Description     string `json:"description"`
	ImageURL        string `json:"image_url"`
	MatchedKeywords string `json:"matched_keywords"`
}

type searchResponse struct {
	Query   string         `json:"query"`
	Results []searchResult `json:"results"`
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	key := "search:" + strings.ToLower(query)
	if e, ok := h.cache.get(key); ok {
		writeHTML(w, r, e)
		return
	}

	resp := searchResponse{Query: query, Results: []searchResult{}}

	if query != "" {
		var embedding []float32
		if h.embedder != nil {
			vecs, err := h.embedder.Embed(r.Context(), []string{query})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if len(vecs) > 0 {
				embedding = vecs[0]
			}
		}

		entries, err := h.entryStore.Search(r.Context(), query, embedding, searchLimit, h.config.SearchMaxDistance)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, e := range entries {
			resp.Results = append(resp.Results, searchResult{
				Title:           e.Title,
				Link:            e.Link,
				Source:          utils.Readable(e.Source),
				Age:             timeAgo(e.CreatedAt),
				Description:     e.Description,
				ImageURL:        e.ImageURL,
				MatchedKeywords: e.MatchedKeywords,
			})
		}
	}

	body, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeHTML(w, r, h.cache.set(key, body))
}
