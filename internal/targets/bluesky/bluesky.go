package bluesky

import (
	"bytes"
	"cartero/internal/components"
	"cartero/internal/platforms"
	"cartero/internal/types"
	"cartero/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/lex/util"
)

type Target struct {
	name      string
	platform  *platforms.BlueskyPlatform
	languages []string
	template  *template.Template
}

func New(name string, languages []string, registry *components.Registry) *Target {
	platformCmp := registry.Get(components.PlatformComponentName).(*components.PlatformComponent)

	tmpl, err := utils.LoadTemplate("templates/bluesky.tmpl")
	if err != nil {
		panic(err.Error())
	}

	return &Target{
		name:      name,
		platform:  platformCmp.Bluesky(),
		languages: languages,
		template:  tmpl,
	}
}

func (t *Target) Name() string {
	return t.name
}

func (t *Target) Initialize(ctx context.Context) error {
	return nil
}

func (t *Target) Publish(ctx context.Context, item *types.Item) (*types.PublishResult, error) {
	var buf bytes.Buffer
	if err := t.template.Execute(&buf, item); err != nil {
		return nil, fmt.Errorf("template execution error: %w", err)
	}

	var output struct {
		Segments []struct {
			Text string `json:"text"`
			URI  string `json:"uri"`
		} `json:"segments"`
		Embed struct {
			URI         string `json:"uri"`
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"embed"`
	}

	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("failed to parse template output: %w", err)
	}

	var text string
	var facets []*bsky.RichtextFacet

	for _, seg := range output.Segments {
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

	post := &bsky.FeedPost{
		CreatedAt: time.Now().Format(time.RFC3339),
		Langs:     t.languages,
		Text:      text,
		Facets:    facets,
	}

	if output.Embed.URI != "" {
		post.Embed = &bsky.FeedPost_Embed{
			EmbedExternal: &bsky.EmbedExternal{
				External: &bsky.EmbedExternal_External{
					Title:       output.Embed.Title,
					Description: output.Embed.Description,
					Uri:         output.Embed.URI,
				},
			},
		}
	}

	resp, err := atproto.RepoCreateRecord(ctx, t.platform.Client(), &atproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       t.platform.Client().Auth.Did,
		Record:     &util.LexiconTypeDecoder{Val: post},
	})

	if err != nil {
		return &types.PublishResult{
			Success:   false,
			Target:    t.name,
			ItemID:    item.ID,
			Timestamp: time.Now(),
			Error:     err,
		}, err
	}

	return &types.PublishResult{
		Success:   true,
		Target:    t.name,
		ItemID:    item.ID,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"uri": resp.Uri,
			"cid": resp.Cid,
		},
	}, nil
}

func (t *Target) Shutdown(ctx context.Context) error {
	return nil
}
