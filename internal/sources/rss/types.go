package rss

type Feed struct {
	URL      string
	Name     string
	MaxItems int
}

type SourceLoader interface {
	Load(value string, maxItems int) ([]Feed, error)
}
