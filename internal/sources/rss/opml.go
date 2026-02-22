package rss

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Body    OPMLBody `xml:"body"`
}

type OPMLBody struct {
	Outlines []OPMLOutline `xml:"outline"`
}

type OPMLOutline struct {
	Title    string        `xml:"title,attr"`
	Text     string        `xml:"text,attr"`
	Type     string        `xml:"type,attr"`
	XMLURL   string        `xml:"xmlUrl,attr"`
	Outlines []OPMLOutline `xml:"outline"`
}

func ParseOPML(data []byte) ([]Feed, error) {
	var opml OPML
	if err := xml.Unmarshal(data, &opml); err != nil {
		return nil, fmt.Errorf("failed to parse OPML: %w", err)
	}

	var feeds []Feed
	extractFeeds(&feeds, opml.Body.Outlines)

	return feeds, nil
}

func extractFeeds(result *[]Feed, outlines []OPMLOutline) {
	for _, outline := range outlines {
		if outline.XMLURL != "" {
			name := outline.Title
			if name == "" {
				name = outline.Text
			}
			if name == "" {
				name = outline.XMLURL
			}

			name = sanitizeName(name)

			*result = append(*result, Feed{
				URL:  outline.XMLURL,
				Name: name,
			})
		}

		if len(outline.Outlines) > 0 {
			extractFeeds(result, outline.Outlines)
		}
	}
}

func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, "&", "and")

	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

func LoadOPMLFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read OPML file: %w", err)
	}
	return data, nil
}

func FetchOPML(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OPML: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch OPML: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OPML response: %w", err)
	}

	return data, nil
}
