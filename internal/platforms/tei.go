package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TEIReranker struct {
	baseURL string
	client  *http.Client
}

func NewTEIReranker(baseURL string) *TEIReranker {
	return &TEIReranker{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type teiRerankRequest struct {
	Query string   `json:"query"`
	Texts []string `json:"texts"`
}

type teiRerankResult struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
}

func (r *TEIReranker) Rerank(ctx context.Context, query string, docs []string) ([]float64, error) {
	if len(docs) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(teiRerankRequest{Query: query, Texts: docs})
	if err != nil {
		return nil, fmt.Errorf("rerank: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", r.baseURL+"/rerank", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("rerank: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rerank: post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("rerank: status %d: %s", resp.StatusCode, string(b))
	}

	var results []teiRerankResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("rerank: decode response: %w", err)
	}

	scores := make([]float64, len(docs))
	for _, res := range results {
		if res.Index >= 0 && res.Index < len(docs) {
			scores[res.Index] = res.Score
		}
	}
	return scores, nil
}
