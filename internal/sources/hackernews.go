package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cartero/internal/core"
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

func (h *HackerNewsSource) Fetch(ctx context.Context) (<-chan *core.Item, <-chan error) {
	itemChan := make(chan *core.Item)
	errChan := make(chan error, 1)

	go func() {
		defer close(itemChan)
		defer close(errChan)

		storyIDs, err := h.fetchStoryIDs(ctx)
		if err != nil {
			errChan <- err
			return
		}

		limit := h.maxItems
		if limit > len(storyIDs) {
			limit = len(storyIDs)
		}

		for i := 0; i < limit; i++ {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				story, err := h.fetchStory(ctx, storyIDs[i])
				if err != nil {
					continue
				}

				item := &core.Item{
					ID:        fmt.Sprintf("hn_%d", story.ID),
					Content:   story,
					Source:    h.name,
					Timestamp: time.Unix(story.Time, 0),
					Metadata: map[string]interface{}{
						"score":      story.Score,
						"author":     story.By,
						"comments":   story.Descendants,
						"story_type": h.storyType,
						"hn_id":      story.ID,
						"title":      story.Title,
						"url":        story.URL,
					},
				}

				select {
				case itemChan <- item:
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				}
			}
		}
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
