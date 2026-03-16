package utils

import (
	"github.com/enetx/surf"
	"net/http"
	"time"
)

type BrowserTransport struct {
	Base    http.RoundTripper
	Headers map[string]string
}

func (t *BrowserTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())

	for key, value := range t.Headers {
		if req2.Header.Get(key) == "" {
			req2.Header.Set(key, value)
		}
	}

	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	return base.RoundTrip(req2)
}

func NewHTTPClient(timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	surfClient := surf.NewClient().
		Builder().
		Impersonate().Firefox().
		Timeout(timeout).
		Session().
		Build().
		Unwrap()

	client := surfClient.Std()

	return client
}
