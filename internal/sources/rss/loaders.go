package rss

type RSSLoader struct{}

func (r *RSSLoader) Load(value string, maxItems int) ([]Feed, error) {
	return []Feed{
		{
			URL:      value,
			Name:     "rss",
			MaxItems: maxItems,
		},
	}, nil
}

type OPMLFileLoader struct{}

func (o *OPMLFileLoader) Load(path string, maxItems int) ([]Feed, error) {
	data, err := LoadOPMLFile(path)
	if err != nil {
		return nil, err
	}

	feeds, err := ParseOPML(data)
	if err != nil {
		return nil, err
	}

	for i := range feeds {
		feeds[i].MaxItems = maxItems
	}

	return feeds, nil
}

type OPMLURLLoader struct{}

func (o *OPMLURLLoader) Load(url string, maxItems int) ([]Feed, error) {
	data, err := FetchOPML(url)
	if err != nil {
		return nil, err
	}

	feeds, err := ParseOPML(data)
	if err != nil {
		return nil, err
	}

	for i := range feeds {
		feeds[i].MaxItems = maxItems
	}

	return feeds, nil
}

func init() {
	RegisterLoader("rss", &RSSLoader{})
	RegisterLoader("opml_file", &OPMLFileLoader{})
	RegisterLoader("opml_url", &OPMLURLLoader{})
}
