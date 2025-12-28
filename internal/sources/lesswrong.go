package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"cartero/internal/core"
)

type LessWrongSource struct {
	name       string
	apiURL     string
	httpClient *http.Client
	maxItems   int
}

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data struct {
		Posts struct {
			Results []LWPost `json:"results"`
		} `json:"posts"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

type LWPost struct {
	ID           string    `json:"_id"`
	Title        string    `json:"title"`
	Slug         string    `json:"slug"`
	BaseScore    int       `json:"baseScore"`
	VoteCount    int       `json:"voteCount"`
	CommentCount int       `json:"commentCount"`
	PostedAt     time.Time `json:"postedAt"`
	URL          string    `json:"url"`
}

func NewLessWrongSource(name string, maxItems int) *LessWrongSource {
	if maxItems == 0 {
		maxItems = 20
	}

	return &LessWrongSource{
		name:       name,
		apiURL:     "https://www.lesswrong.com/graphql",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		maxItems:   maxItems,
	}
}

func (l *LessWrongSource) Name() string {
	return l.name
}

func (l *LessWrongSource) Initialize(ctx context.Context) error {
	log.Printf("LessWrong source %s: initializing (view=curated, max_items=%d)", l.name, l.maxItems)
	return nil
}

func (l *LessWrongSource) Fetch(ctx context.Context) (<-chan *core.Item, <-chan error) {
	itemChan := make(chan *core.Item)
	errChan := make(chan error, 1)

	go func() {
		defer close(itemChan)
		defer close(errChan)

		log.Printf("LessWrong source %s: fetching posts", l.name)
		posts, err := l.fetchPosts(ctx)
		if err != nil {
			log.Printf("LessWrong source %s: error fetching posts: %v", l.name, err)
			errChan <- err
			return
		}

		log.Printf("LessWrong source %s: retrieved %d posts", l.name, len(posts))

		for i, post := range posts {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				// Build full URL if not present
				postURL := post.URL
				if postURL == "" {
					postURL = fmt.Sprintf("https://www.lesswrong.com/posts/%s/%s", post.ID, post.Slug)
				}

				item := &core.Item{
					ID:        fmt.Sprintf("lw_%s", post.ID),
					Source:    l.name,
					Timestamp: post.PostedAt,
					Content:   post,
					Metadata: map[string]interface{}{
						"score":         post.BaseScore,
						"comments":      postURL,
						"comment_count": post.CommentCount,
						"vote_count":    post.VoteCount,
						"lw_id":         post.ID,
						"title":         post.Title,
						"url":           postURL,
					},
				}

				select {
				case itemChan <- item:
					log.Printf("LessWrong source %s: sent item %d/%d (id=%s, score=%d)",
						l.name, i+1, len(posts), post.ID, post.BaseScore)
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				}
			}
		}

		log.Printf("LessWrong source %s: finished fetching all items", l.name)
	}()

	return itemChan, errChan
}

func (l *LessWrongSource) fetchPosts(ctx context.Context) ([]LWPost, error) {
	query := fmt.Sprintf(`
		query {
			posts(input: {
				terms: {
					view: "curated"
					limit: %d
				}
			}) {
				results {
					_id
					title
					slug
					baseScore
					voteCount
					commentCount
					postedAt
					url
				}
			}
		}
	`, l.maxItems)

	gqlReq := GraphQLRequest{
		Query: query,
	}

	body, err := json.Marshal(gqlReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", l.apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Cartero/1.0")

	log.Printf("LessWrong source %s: sending GraphQL request: %s", l.name, string(body))

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("LessWrong source %s: GraphQL error (status %d): %s", l.name, resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var gqlResp GraphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	return gqlResp.Data.Posts.Results, nil
}

func (l *LessWrongSource) Shutdown(ctx context.Context) error {
	log.Printf("LessWrong source %s: shutting down", l.name)
	return nil
}
