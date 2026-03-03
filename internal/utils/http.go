package utils

import (
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

func GetDefaultBrowserHeaders() map[string]string {
	return map[string]string{
		"sec-ch-ua":                 `"Chromium";v="142", "Google Chrome";v="142", "Not_A Brand";v="99"`,
		"sec-ch-ua-mobile":          "?0",
		"sec-ch-ua-platform":        `"macOS"`,
		"upgrade-insecure-requests": "1",
		"user-agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36",
		"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"sec-fetch-site":            "none",
		"sec-fetch-mode":            "navigate",
		"sec-fetch-user":            "?1",
		"sec-fetch-dest":            "document",
		"accept-encoding":           "gzip, deflate, br, zstd",
		"accept-language":           "en-US,en;q=0.9",
		"priority":                  "u=0, i",
	}
}

func NewHTTPClient(timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &BrowserTransport{
			Base:    http.DefaultTransport,
			Headers: GetDefaultBrowserHeaders(),
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}

func BrowserHeadersModifier() func(*http.Request) {
	headers := GetDefaultBrowserHeaders()
	return func(req *http.Request) {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}
}

func NewHTTPClientWithHeaders(timeout time.Duration, customHeaders map[string]string) *http.Client {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	headers := GetDefaultBrowserHeaders()

	for key, value := range customHeaders {
		headers[key] = value
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &BrowserTransport{
			Base:    http.DefaultTransport,
			Headers: headers,
		},
	}
}
