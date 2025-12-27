package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"cartero/internal/core"
)

type LobstersSource struct {
	name              string
	httpClient        *http.Client
	maxItems          int
	sortBy            string
	includeCategories []string
	excludeCategories []string
}

type LobstersPost struct {
	CreatedAt    string   `json:"created_at"`
	Title        string   `json:"title"`
	URL          string   `json:"url"`
	Score        int      `json:"score"`
	CommentCount int      `json:"comment_count"`
	CommentsURL  string   `json:"comments_url"`
	Tags         []string `json:"tags"`
	ShortID      string   `json:"short_id"`
	Submitter    string   `json:"submitter_user"`
}

func NewLobstersSource(name string, maxItems int, sortBy string, includeCategories, excludeCategories []string) *LobstersSource {
	if maxItems == 0 {
		maxItems = 50
	}

	if sortBy == "" {
		sortBy = "hot"
	}

	return &LobstersSource{
		name:              name,
		httpClient:        &http.Client{Timeout: 30 * time.Second},
		maxItems:          maxItems,
		sortBy:            sortBy,
		includeCategories: includeCategories,
		excludeCategories: excludeCategories,
	}
}

func (l *LobstersSource) Name() string {
	return l.name
}

func (l *LobstersSource) Initialize(ctx context.Context) error {
	log.Printf("Lobsters source %s: initializing (sort_by=%s, max_items=%d)", l.name, l.sortBy, l.maxItems)
	return nil
}

func (l *LobstersSource) Fetch(ctx context.Context) (<-chan *core.Item, <-chan error) {
	itemChan := make(chan *core.Item)
	errChan := make(chan error, 1)

	go func() {
		defer close(itemChan)
		defer close(errChan)

		log.Printf("Lobsters source %s: fetching posts", l.name)
		posts, err := l.fetchPosts(ctx)
		if err != nil {
			log.Printf("Lobsters source %s: error fetching posts: %v", l.name, err)
			errChan <- err
			return
		}

		log.Printf("Lobsters source %s: retrieved %d posts", l.name, len(posts))

		limit := l.maxItems
		if limit > len(posts) {
			limit = len(posts)
		}

		log.Printf("Lobsters source %s: processing %d posts", l.name, limit)

		for i := 0; i < limit; i++ {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				post := posts[i]

				if !l.shouldIncludePost(post) {
					continue
				}

				createdAt, _ := time.Parse(time.RFC3339, post.CreatedAt)

				item := &core.Item{
					ID:        fmt.Sprintf("lobsters_%s", post.ShortID),
					Content:   post,
					Source:    l.name,
					Timestamp: createdAt,
					Metadata: map[string]interface{}{
						"title":         post.Title,
						"url":           post.URL,
						"link":          post.URL,
						"score":         post.Score,
						"author":        post.Submitter,
						"comments":      post.CommentsURL,
						"comment_count": post.CommentCount,
						"tags":          post.Tags,
						"category":      strings.Join(post.Tags, ","),
					},
				}

				select {
				case itemChan <- item:
					log.Printf("Lobsters source %s: sent item %d/%d (id=%s, score=%d)",
						l.name, i+1, limit, post.ShortID, post.Score)
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				}
			}
		}

		log.Printf("Lobsters source %s: finished processing all items", l.name)
	}()

	return itemChan, errChan
}

func (l *LobstersSource) fetchPosts(ctx context.Context) ([]LobstersPost, error) {
	feedURL := l.buildFeedURL()

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Cartero-Bot/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var posts []LobstersPost
	if err := json.Unmarshal(body, &posts); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return posts, nil
}

func (l *LobstersSource) buildFeedURL() string {
	baseURL := "https://lobste.rs/"

	sortPath := l.sortBy
	if sortPath == "hot" {
		sortPath = "hottest"
	} else if sortPath == "new" {
		sortPath = "newest"
	}

	if len(l.includeCategories) > 0 {
		tags := strings.Join(l.includeCategories, ",")
		return fmt.Sprintf("%st/%s.json", baseURL, tags)
	}

	return fmt.Sprintf("%s%s.json", baseURL, sortPath)
}

func (l *LobstersSource) shouldIncludePost(post LobstersPost) bool {
	if len(l.excludeCategories) > 0 {
		for _, postTag := range post.Tags {
			for _, excluded := range l.excludeCategories {
				if strings.EqualFold(postTag, excluded) {
					return false
				}
			}
		}
	}

	return true
}

func (l *LobstersSource) Shutdown(ctx context.Context) error {
	log.Printf("Lobsters source %s: shutting down", l.name)
	l.httpClient.CloseIdleConnections()
	return nil
}
