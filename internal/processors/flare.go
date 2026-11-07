package processors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cartero/internal/types"
	"cartero/internal/utils"
)

const (
	defaultFlareTimeout = 60 * time.Second
	flareGracePeriod    = 30 * time.Second
	flareMaxBodyBytes   = 2 * utils.MaxArticleBytes
	flareSessionName    = "cartero"
)

var errFlareSession = errors.New("flaresolverr session lost")

type FlareSolverrExtractor struct {
	endpoint   *url.URL
	session    string
	maxTimeout time.Duration
	client     *http.Client
	sem        chan struct{}
}

func NewFlareSolverrExtractor(baseURL string, maxTimeout time.Duration) (*FlareSolverrExtractor, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("flaresolverr url: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("flaresolverr url %q: scheme and host are required", baseURL)
	}
	if maxTimeout <= 0 {
		maxTimeout = defaultFlareTimeout
	}
	return &FlareSolverrExtractor{
		endpoint:   base.JoinPath("v1"),
		maxTimeout: maxTimeout,
		client:     &http.Client{},
		sem:        make(chan struct{}, 1),
	}, nil
}

type flareRequest struct {
	Cmd        string `json:"cmd"`
	URL        string `json:"url,omitempty"`
	Session    string `json:"session,omitempty"`
	MaxTimeout int    `json:"maxTimeout,omitempty"`
}

type flareResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Session  string `json:"session"`
	Solution struct {
		Status   int    `json:"status"`
		Response string `json:"response"`
	} `json:"solution"`
}

func (f *FlareSolverrExtractor) Extract(ctx context.Context, u *url.URL, _ time.Duration) (*types.Article, error) {
	ctx, cancel := context.WithTimeout(ctx, f.maxTimeout+flareGracePeriod)
	defer cancel()

	select {
	case f.sem <- struct{}{}:
		defer func() { <-f.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	resp, err := f.fetch(ctx, u)
	if errors.Is(err, errFlareSession) {
		f.session = ""
		resp, err = f.fetch(ctx, u)
	}
	if err != nil {
		return nil, err
	}

	if resp.Solution.Status != http.StatusOK {
		return nil, fmt.Errorf("flaresolverr fetch %s: status %d", u.Hostname(), resp.Solution.Status)
	}
	if strings.TrimSpace(resp.Solution.Response) == "" {
		return nil, fmt.Errorf("flaresolverr returned an empty page")
	}

	return utils.ParseArticle([]byte(resp.Solution.Response), u)
}

func (f *FlareSolverrExtractor) fetch(ctx context.Context, u *url.URL) (*flareResponse, error) {
	if err := f.ensureSession(ctx); err != nil {
		return nil, err
	}
	return f.call(ctx, flareRequest{
		Cmd:        "request.get",
		URL:        u.String(),
		Session:    f.session,
		MaxTimeout: int(f.maxTimeout.Milliseconds()),
	})
}

func (f *FlareSolverrExtractor) ensureSession(ctx context.Context) error {
	if f.session != "" {
		return nil
	}

	resp, err := f.call(ctx, flareRequest{Cmd: "sessions.create", Session: flareSessionName})
	if err != nil {
		if !errors.Is(err, errFlareSession) {
			return err
		}
		if _, derr := f.call(ctx, flareRequest{Cmd: "sessions.destroy", Session: flareSessionName}); derr != nil {
			return err
		}
		if resp, err = f.call(ctx, flareRequest{Cmd: "sessions.create", Session: flareSessionName}); err != nil {
			return err
		}
	}

	f.session = resp.Session
	if f.session == "" {
		f.session = flareSessionName
	}
	return nil
}

func (f *FlareSolverrExtractor) call(ctx context.Context, req flareRequest) (*flareResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, f.endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	hreq.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(hreq)
	if err != nil {
		return nil, fmt.Errorf("flaresolverr request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("flaresolverr returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, flareMaxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("flaresolverr read failed: %w", err)
	}

	var parsed flareResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("flaresolverr response: %w", err)
	}
	if parsed.Status != "ok" {
		if strings.Contains(strings.ToLower(parsed.Message), "session") {
			return nil, fmt.Errorf("%w: %s", errFlareSession, parsed.Message)
		}
		return nil, fmt.Errorf("flaresolverr: %s", parsed.Message)
	}

	return &parsed, nil
}
