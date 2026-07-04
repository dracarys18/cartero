package rss

import "net/url"

type Feed struct {
	URL      *url.URL
	Name     string
	MaxItems int
}

type SourceLoader interface {
	Load(value string, maxItems int) ([]Feed, error)
}
