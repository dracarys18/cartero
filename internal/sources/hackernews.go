package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cartero/internal/types"
)

type HackerNewsSource struct {
	name       string
	apiURL     string
	httpClient *http.Client
	maxItems   int
	storyType  string
}

type HNStory struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Score       int    `json:"score"`
	By          string `json:"by"`
	Time        int64  `json:"time"`
	Descendants int    `json:"descendants"`
	Type        string `json:"type"`
	Text        string `json:"text"`
}

func NewHackerNewsSource(name string, storyType string, maxItems int) *HackerNewsSource {
	if storyType == "" {
		storyType = "topstories"
	}
	if maxItems == 0 {
		maxItems = 30
	}

	return &HackerNewsSource{
		name:       name,
		apiURL:     "https://hacker-news.firebaseio.com/v0",
		httpClient: &http.Client{Timeout: 10 * time.Second},
		maxItems:   maxItems,
		storyType:  storyType,
	}
}

func (h *HackerNewsSource) Name() string {
	return h.name
}

func (h *HackerNewsSource) Initialize(ctx context.Context) error {
	return nil
}

func (h *HackerNewsSource) Fetch(ctx context.Context, state types.StateAccessor) (<-chan *types.Item, <-chan error) {
	itemChan := make(chan *types.Item)
	errChan := make(chan error, 1)

	go func() {
		defer close(itemChan)
		defer close(errChan)

		logger := state.GetLogger()

		storyIDs, err := h.fetchStoryIDs(ctx)
		if err != nil {
			logger.Error("HackerNews source error fetching story IDs", "source", h.name, "error", err)
			errChan <- err
			return
		}

		logger.Debug("HackerNews source retrieved story IDs", "source", h.name, "count", len(storyIDs))

		limit := h.maxItems
		if limit > len(storyIDs) {
			limit = len(storyIDs)
		}

		logger.Debug("HackerNews source fetching stories", "source", h.name, "limit", limit)

		for i := 0; i < limit; i++ {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				story, err := h.fetchStory(ctx, storyIDs[i])
				if err != nil {
					logger.Warn("HackerNews source error fetching story", "source", h.name, "story_id", storyIDs[i], "error", err)
					continue
				}

				item := &types.Item{
					ID:        fmt.Sprintf("hn_%d", story.ID),
					Source:    h.name,
					Timestamp: time.Unix(story.Time, 0),
					Content:   story,
					Metadata: map[string]interface{}{
						"score":         story.Score,
						"author":        story.By,
						"comments":      fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID),
						"comment_count": story.Descendants,
						"story_type":    h.storyType,
						"hn_id":         story.ID,
						"title":         story.Title,
						"url":           story.URL,
					},
				}

				select {
				case itemChan <- item:
					logger.Debug("HackerNews source sent item", "source", h.name, "index", i+1, "limit", limit, "story_id", story.ID, "score", story.Score)
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				}
			}
		}

		logger.Debug("HackerNews source finished fetching all items", "source", h.name)
	}()

	return itemChan, errChan
}

func (h *HackerNewsSource) fetchStoryIDs(ctx context.Context) ([]int64, error) {
	url := fmt.Sprintf("%s/%s.json", h.apiURL, h.storyType)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch story IDs: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var storyIDs []int64
	if err := json.Unmarshal(body, &storyIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal story IDs: %w", err)
	}

	return storyIDs, nil
}

func (h *HackerNewsSource) fetchStory(ctx context.Context, id int64) (*HNStory, error) {
	url := fmt.Sprintf("%s/item/%d.json", h.apiURL, id)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch story: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var story HNStory
	if err := json.Unmarshal(body, &story); err != nil {
		return nil, fmt.Errorf("failed to unmarshal story: %w", err)
	}

	return &story, nil
}

func (h *HackerNewsSource) Shutdown(ctx context.Context) error {
	return nil
}
