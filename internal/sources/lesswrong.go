package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cartero/internal/types"
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
	return nil
}

func (l *LessWrongSource) Publish(ctx context.Context, state types.StateAccessor) error {
	logger := state.GetLogger()
	q := state.GetQueue()
	stream := q.SourceStream()

	posts, err := l.fetchPosts(ctx)
	if err != nil {
		logger.Error("LessWrong source error fetching posts", "source", l.name, "error", err)
		return err
	}

	logger.Debug("LessWrong source retrieved posts", "source", l.name, "count", len(posts))

	for i, post := range posts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		postURL := post.URL
		if postURL == "" {
			postURL = fmt.Sprintf("https://www.lesswrong.com/posts/%s/%s", post.ID, post.Slug)
		}

		item := &types.Item{
			ID:        fmt.Sprintf("lw_%s", post.ID),
			Title:     post.Title,
			URL:       postURL,
			Source:    l.name,
			Route:     l.name,
			Timestamp: post.PostedAt,
			Content:   post,
			Metadata: map[string]interface{}{
				"score":         post.BaseScore,
				"comments":      postURL,
				"comment_count": post.CommentCount,
				"vote_count":    post.VoteCount,
				"lw_id":         post.ID,
				"title":         post.Title,
			},
		}

		if err := q.Publish(ctx, stream, types.Envelope{Item: item}); err != nil {
			return err
		}
		logger.Debug("LessWrong source published item", "source", l.name, "index", i+1, "total", len(posts), "post_id", post.ID, "score", post.BaseScore)
	}

	logger.Debug("LessWrong source finished fetching all items", "source", l.name)
	return nil
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

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
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
	return nil
}
