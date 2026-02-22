package bluesky

import (
	"bytes"
	"cartero/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
)

type Post struct {
	Segments []Segment  `json:"segments"`
	Embed    *EmbedData `json:"embed,omitempty"`
}

type Segment struct {
	Text string `json:"text"`
	URI  string `json:"uri,omitempty"`
}

type EmbedData struct {
	URI          string `json:"uri"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

type RichText struct {
	Text   string
	Facets []*bsky.RichtextFacet
}

type BlueskyPost struct {
	Post       *bsky.FeedPost
	ImageURL   string
	XRPCClient *xrpc.Client
}

func (p *Post) TryFrom(templateOutput []byte) error {
	if err := json.Unmarshal(templateOutput, p); err != nil {
		return fmt.Errorf("bluesky: failed to unmarshal template output to Post: %w", err)
	}
	return nil
}

func (p *Post) From(item *types.Item) {
	if title, ok := item.Metadata["title"].(string); ok && title != "" {
		p.Segments = append(p.Segments, Segment{Text: title})
		p.Segments = append(p.Segments, Segment{Text: "\n"})
	}

	if comments, ok := item.Metadata["comments"].(string); ok && comments != "" {
		p.Segments = append(p.Segments, Segment{
			Text: "Discussion",
			URI:  comments,
		})
		p.Segments = append(p.Segments, Segment{Text: " · "})
	}

	if url, ok := item.Metadata["url"].(string); ok && url != "" {
		p.Segments = append(p.Segments, Segment{
			Text: "Read More",
			URI:  url,
		})

		title, _ := item.Metadata["title"].(string)
		p.Embed = &EmbedData{
			URI:   url,
			Title: title,
		}

		if item.TextContent != nil {
			p.Embed.Description = item.TextContent.Description
		}
	}
}

func (p *Post) Into() RichText {
	var text string
	var facets []*bsky.RichtextFacet

	for _, seg := range p.Segments {
		if seg.Text == "" {
			continue
		}

		start := int64(len(text))
		text += seg.Text
		end := int64(len(text))

		if seg.URI != "" {
			facets = append(facets, &bsky.RichtextFacet{
				Index: &bsky.RichtextFacet_ByteSlice{
					ByteStart: start,
					ByteEnd:   end,
				},
				Features: []*bsky.RichtextFacet_Features_Elem{
					{
						RichtextFacet_Link: &bsky.RichtextFacet_Link{
							Uri: seg.URI,
						},
					},
				},
			})
		}
	}

	return RichText{
		Text:   text,
		Facets: facets,
	}
}

func (e *EmbedData) TryInto() (*bsky.EmbedExternal_External, error) {
	if e.URI == "" {
		return nil, fmt.Errorf("bluesky: embed URI cannot be empty")
	}

	description := e.Description
	if len(description) > 300 {
		description = description[:297] + "..."
	}

	return &bsky.EmbedExternal_External{
		Uri:         e.URI,
		Title:       e.Title,
		Description: description,
	}, nil
}

func BuildPost(richText RichText, embedExternal *bsky.EmbedExternal_External, languages []string) *bsky.FeedPost {
	post := &bsky.FeedPost{
		CreatedAt: time.Now().Format(time.RFC3339),
		Langs:     languages,
		Text:      richText.Text,
		Facets:    richText.Facets,
	}

	if embedExternal != nil {
		post.Embed = &bsky.FeedPost_Embed{
			EmbedExternal: &bsky.EmbedExternal{
				External: embedExternal,
			},
		}
	}

	return post
}

func UploadBlob(ctx context.Context, c *xrpc.Client, imageURL string) (*util.LexBlob, error) {
	if imageURL == "" {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")

	resp, reqErr := http.DefaultClient.Do(req)
	if reqErr != nil {
		return nil, reqErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch image: %s", resp.Status)
	}

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}

	blobResp, blobErr := atproto.RepoUploadBlob(ctx, c, bytes.NewReader(data))
	if blobErr != nil {
		return nil, blobErr
	}

	return blobResp.Blob, nil
}

func AttachThumbnail(post *bsky.FeedPost, blob *util.LexBlob) {
	if post.Embed != nil && post.Embed.EmbedExternal != nil && blob != nil {
		post.Embed.EmbedExternal.External.Thumb = blob
	}
}
