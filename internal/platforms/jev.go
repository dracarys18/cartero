package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultJevBaseURL    = "https://api.typesafe.ai"
	defaultJevModel      = "jev-latest"
	jevMaxRetries        = 2
	jevBackoffInitial    = 500 * time.Millisecond
	jevMaxRetryAfter     = 10 * time.Second
	jevRequestTimeout    = 15 * time.Second
	jevErrorBodyMaxBytes = 1024
)

type JevPlatform struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewJevPlatform(baseURL, apiKey, model string) *JevPlatform {
	if baseURL == "" {
		baseURL = defaultJevBaseURL
	}
	if model == "" {
		model = defaultJevModel
	}
	return &JevPlatform{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: jevRequestTimeout},
	}
}

type JevQuestion struct {
	Type         string `json:"type"`
	Instructions any    `json:"instructions,omitempty"`
	Criteria     any    `json:"criteria,omitempty"`
}

func JevChoice(instructions any, criteria map[string]any) JevQuestion {
	return JevQuestion{Type: "choice", Instructions: instructions, Criteria: criteria}
}

type JevAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type JevResponse struct {
	Model   string               `json:"model"`
	Answers map[string]JevAnswer `json:"answers"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type jevRequest struct {
	Model     string                 `json:"model"`
	State     any                    `json:"state"`
	Questions map[string]JevQuestion `json:"questions"`
}

type JevAPIError struct {
	StatusCode int
	Body       string
	retryAfter time.Duration
}

func (e *JevAPIError) Error() string {
	return fmt.Sprintf("jev: status %d: %s", e.StatusCode, e.Body)
}

func (e *JevAPIError) retryable() bool {
	return e.StatusCode == http.StatusRequestTimeout || e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= 500
}

func (p *JevPlatform) SystemOne(ctx context.Context, state any, questions map[string]JevQuestion) (*JevResponse, error) {
	body, err := json.Marshal(jevRequest{Model: p.model, State: state, Questions: questions})
	if err != nil {
		return nil, fmt.Errorf("jev: marshal request: %w", err)
	}

	backoff := jevBackoffInitial
	for attempt := 0; ; attempt++ {
		resp, err := p.do(ctx, body)
		if err == nil {
			return resp, nil
		}
		if attempt >= jevMaxRetries || ctx.Err() != nil {
			return nil, err
		}

		wait := backoff
		var apiErr *JevAPIError
		if errors.As(err, &apiErr) {
			if !apiErr.retryable() {
				return nil, err
			}
			if apiErr.retryAfter > 0 {
				wait = apiErr.retryAfter
			}
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		backoff *= 2
	}
}

func (p *JevPlatform) do(ctx context.Context, body []byte) (*JevResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/systemone", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("jev: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jev: post: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, jevErrorBodyMaxBytes))
		return nil, &JevAPIError{
			StatusCode: resp.StatusCode,
			Body:       string(b),
			retryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
	}

	var result JevResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("jev: decode response: %w", err)
	}
	return &result, nil
}

func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs <= 0 {
		return 0
	}
	return min(time.Duration(secs)*time.Second, jevMaxRetryAfter)
}
