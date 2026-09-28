package handler

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"time"

	"cartero/internal/storage"
	utils "cartero/internal/utils/string"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

const (
	topicDomain  = "topic"
	sourceDomain = "source"
)

var markdown = goldmark.New(goldmark.WithExtensions(extension.GFM))

func renderHTML(md string) string {
	if md == "" {
		return ""
	}
	var buf bytes.Buffer
	if err := markdown.Convert([]byte(md), &buf); err != nil {
		return ""
	}
	return buf.String()
}

type cdata struct {
	Value string `xml:",cdata"`
}

type mediaContent struct {
	URL    string `xml:"url,attr"`
	Medium string `xml:"medium,attr"`
}

func imageMedia(url string) *mediaContent {
	if url == "" {
		return nil
	}
	return &mediaContent{URL: url, Medium: "image"}
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Content string     `xml:"xmlns:content,attr"`
	DC      string     `xml:"xmlns:dc,attr"`
	Media   string     `xml:"xmlns:media,attr"`
	Atom    string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Self          *atomLink `xml:"atom:link"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string        `xml:"title"`
	Link        string        `xml:"link,omitempty"`
	GUID        rssGUID       `xml:"guid"`
	Description string        `xml:"description,omitempty"`
	Content     *cdata        `xml:"content:encoded"`
	Creator     string        `xml:"dc:creator,omitempty"`
	PubDate     string        `xml:"pubDate,omitempty"`
	Categories  []rssCategory `xml:"category"`
	Media       *mediaContent `xml:"media:content"`
}

type rssGUID struct {
	IsPermaLink bool   `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type rssCategory struct {
	Domain string `xml:"domain,attr"`
	Value  string `xml:",chardata"`
}

type atomFeed struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Media   string      `xml:"xmlns:media,attr"`
	Title   string      `xml:"title"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Links   []atomLink  `xml:"link"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title      string         `xml:"title"`
	ID         string         `xml:"id"`
	Link       atomLink       `xml:"link"`
	Published  string         `xml:"published"`
	Updated    string         `xml:"updated"`
	Author     *atomPerson    `xml:"author"`
	Summary    *atomText      `xml:"summary"`
	Content    *atomText      `xml:"content"`
	Categories []atomCategory `xml:"category"`
	Media      *mediaContent  `xml:"media:content"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

type atomPerson struct {
	Name string `xml:"name"`
}

type atomText struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type atomCategory struct {
	Term   string `xml:"term,attr"`
	Scheme string `xml:"scheme,attr"`
}

type jsonFeed struct {
	Version     string         `json:"version"`
	Title       string         `json:"title"`
	HomePageURL string         `json:"home_page_url,omitempty"`
	FeedURL     string         `json:"feed_url,omitempty"`
	Items       []jsonFeedItem `json:"items"`
}

type jsonFeedItem struct {
	ID            string          `json:"id"`
	URL           string          `json:"url,omitempty"`
	Title         string          `json:"title"`
	Summary       string          `json:"summary,omitempty"`
	ContentHTML   string          `json:"content_html,omitempty"`
	ContentText   string          `json:"content_text,omitempty"`
	Image         string          `json:"image,omitempty"`
	DatePublished *time.Time      `json:"date_published,omitempty"`
	Authors       []jsonAuthor    `json:"authors,omitempty"`
	Tags          []string        `json:"tags,omitempty"`
	Cartero       jsonCarteroMeta `json:"_cartero"`
}

type jsonAuthor struct {
	Name string `json:"name"`
}

type jsonCarteroMeta struct {
	Source         string    `json:"source"`
	ReadingMinutes int       `json:"reading_minutes,omitempty"`
	AddedAt        time.Time `json:"added_at"`
}

func (h *Handler) feedURL(path string) string {
	return h.config.SiteURL + path
}

func (h *Handler) encodeRSS(entries []storage.FeedEntry) ([]byte, error) {
	items := make([]rssItem, 0, len(entries))
	for _, e := range entries {
		item := rssItem{
			Title:       e.Title,
			Link:        e.Link,
			GUID:        rssGUID{Value: e.ID},
			Description: e.Description,
			Creator:     e.Author,
			Categories:  []rssCategory{{Domain: sourceDomain, Value: utils.Readable(e.Source)}},
			Media:       imageMedia(e.ImageURL),
		}
		if content := renderHTML(e.Content); content != "" {
			item.Content = &cdata{Value: content}
		}
		if !e.PublishedAt.IsZero() {
			item.PubDate = e.PublishedAt.UTC().Format(time.RFC1123Z)
		}
		if e.MatchedKeywords != "" {
			item.Categories = append(item.Categories, rssCategory{Domain: topicDomain, Value: e.MatchedKeywords})
		}
		items = append(items, item)
	}

	feed := rssFeed{
		Version: "2.0",
		Content: "http://purl.org/rss/1.0/modules/content/",
		DC:      "http://purl.org/dc/elements/1.1/",
		Media:   "http://search.yahoo.com/mrss/",
		Atom:    "http://www.w3.org/2005/Atom",
		Channel: rssChannel{
			Title:         h.config.SiteName,
			Link:          h.config.SiteURL,
			Description:   h.config.SiteDescription,
			Self:          &atomLink{Href: h.feedURL("/feed.rss"), Rel: "self", Type: "application/rss+xml"},
			LastBuildDate: time.Now().UTC().Format(time.RFC1123Z),
			Items:         items,
		},
	}
	return encodeXML(feed)
}

func (h *Handler) encodeAtom(entries []storage.FeedEntry) ([]byte, error) {
	updated := time.Now().UTC()
	out := make([]atomEntry, 0, len(entries))
	for _, e := range entries {
		published := e.PublishedAt
		if published.IsZero() {
			published = e.CreatedAt
		}
		entry := atomEntry{
			Title:      e.Title,
			ID:         "urn:cartero:" + e.ID,
			Link:       atomLink{Href: e.Link, Rel: "alternate"},
			Published:  published.UTC().Format(time.RFC3339),
			Updated:    e.CreatedAt.UTC().Format(time.RFC3339),
			Categories: []atomCategory{{Term: utils.Readable(e.Source), Scheme: sourceDomain}},
			Media:      imageMedia(e.ImageURL),
		}
		if e.Author != "" {
			entry.Author = &atomPerson{Name: e.Author}
		}
		if e.Description != "" {
			entry.Summary = &atomText{Type: "text", Value: e.Description}
		}
		if content := renderHTML(e.Content); content != "" {
			entry.Content = &atomText{Type: "html", Value: content}
		}
		if e.MatchedKeywords != "" {
			entry.Categories = append(entry.Categories, atomCategory{Term: e.MatchedKeywords, Scheme: topicDomain})
		}
		out = append(out, entry)
	}

	feed := atomFeed{
		Media:   "http://search.yahoo.com/mrss/",
		Title:   h.config.SiteName,
		ID:      h.feedURL("/feed.atom"),
		Updated: updated.Format(time.RFC3339),
		Links: []atomLink{
			{Href: h.config.SiteURL, Rel: "alternate"},
			{Href: h.feedURL("/feed.atom"), Rel: "self", Type: "application/atom+xml"},
		},
		Entries: out,
	}
	return encodeXML(feed)
}

func (h *Handler) encodeJSON(entries []storage.FeedEntry) ([]byte, error) {
	feed := jsonFeed{
		Version:     "https://jsonfeed.org/version/1.1",
		Title:       h.config.SiteName,
		HomePageURL: h.config.SiteURL,
		Items:       make([]jsonFeedItem, 0, len(entries)),
	}
	if h.config.SiteURL != "" {
		feed.FeedURL = h.feedURL("/feed.json")
	}

	for _, e := range entries {
		item := jsonFeedItem{
			ID:          e.ID,
			URL:         e.Link,
			Title:       e.Title,
			Summary:     e.Description,
			ContentHTML: renderHTML(e.Content),
			ContentText: e.Content,
			Image:       e.ImageURL,
			Cartero: jsonCarteroMeta{
				Source:         utils.Readable(e.Source),
				ReadingMinutes: readingMinutes(e.Content),
				AddedAt:        e.CreatedAt,
			},
		}
		if !e.PublishedAt.IsZero() {
			item.DatePublished = &e.PublishedAt
		}
		if e.Author != "" {
			item.Authors = []jsonAuthor{{Name: e.Author}}
		}
		if e.MatchedKeywords != "" {
			item.Tags = []string{e.MatchedKeywords}
		}
		feed.Items = append(feed.Items, item)
	}
	return json.Marshal(feed)
}

func encodeXML(v any) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	if err := xml.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
