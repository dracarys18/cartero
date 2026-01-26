package bluesky

import (
	"bytes"
	"cartero/internal/components"
	"cartero/internal/platforms"
	"cartero/internal/types"
	"cartero/internal/utils"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
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

	var post Post
	if err := post.TryFrom(buf.Bytes()); err != nil {
		return nil, err
	}

	richText := post.Into()

	var embedExternal *bsky.EmbedExternal_External
	if post.Embed != nil {
		var err error
		embedExternal, err = post.Embed.TryInto()
		if err != nil {
			return nil, err
		}
	}

	bskyPost := BuildPost(richText, embedExternal, t.languages)

	var resp *atproto.RepoCreateRecord_Output
	err := t.platform.Do(ctx, func(c *xrpc.Client) error {
		imgURL := post.Embed.ThumbnailURL
		if imgURL != "" {
			blob, blobErr := UploadBlob(ctx, c, imgURL)
			if blobErr == nil {
				AttachThumbnail(bskyPost, blob)
			}
		}

		var err error
		resp, err = atproto.RepoCreateRecord(ctx, c, &atproto.RepoCreateRecord_Input{
			Collection: "app.bsky.feed.post",
			Repo:       c.Auth.Did,
			Record:     &util.LexiconTypeDecoder{Val: bskyPost},
		})
		return err
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
