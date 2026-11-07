package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"cartero/internal/types"
	strutils "cartero/internal/utils/string"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
	"github.com/enetx/g"
	"github.com/enetx/surf"
	"github.com/markusmobius/go-trafilatura"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func GetArticle(ctx context.Context, u *url.URL, timeout time.Duration, resolver string) (*types.Article, error) {
	if u == nil || u.String() == "" {
		return nil, fmt.Errorf("URL is empty")
	}

	builder := surf.NewClient().
		Builder().
		Impersonate().Firefox().
		Timeout(timeout).
		Session()
	if resolver != "" {
		builder = builder.DNS(g.String(resolver))
	}
	surfClient := builder.Build().Unwrap()

	client := surfClient.Std()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %d", u.Host, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxArticleBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if provider, blocked := BlockedByAntibot(resp.StatusCode, resp.Header, string(body), u.String()); blocked {
		return nil, &AntibotError{Provider: provider}
	}

	return ParseArticle(body, u)
}

type AntibotError struct {
	Provider string
}

func (e *AntibotError) Error() string {
	return fmt.Sprintf("blocked by %s", e.Provider)
}

func ParseArticle(body []byte, u *url.URL) (*types.Article, error) {
	result, err := trafilatura.Extract(bytes.NewReader(body), trafilatura.Options{
		OriginalURL:     u,
		EnableFallback:  true,
		ExcludeComments: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to extract content: %w", err)
	}

	promoteCodeBlocks(result.ContentNode)

	var buf bytes.Buffer
	if err := html.Render(&buf, result.ContentNode); err != nil {
		return nil, fmt.Errorf("failed to render content: %w", err)
	}
	converter := md.NewConverter(u.Hostname(), true, nil)
	converter.Use(plugin.GitHubFlavored())
	markdown, err := converter.ConvertString(buf.String())
	if err != nil {
		return nil, fmt.Errorf("failed to convert content to markdown: %w", err)
	}
	markdown = stripInlineImages(markdown)
	if isFullDocument(markdown) {
		return nil, fmt.Errorf("extraction returned a full HTML document")
	}
	if !looksLikeArticle(markdown) {
		return nil, fmt.Errorf("extraction returned no article content")
	}

	return &types.Article{
		Text:        markdown,
		Image:       result.Metadata.Image,
		Description: strutils.Clean(result.Metadata.Description),
	}, nil
}

const MaxArticleBytes = 8 << 20

const minArticleWords = 120

func looksLikeArticle(text string) bool {
	return len(strings.Fields(text)) >= minArticleWords
}

// Inline base64 images are single multi-kilobyte tokens: they bloat the row,
// blow past Postgres' 2047-character lexeme limit and bloat the feed payload.
var (
	inlineDataImage  = regexp.MustCompile(`!\[[^\]]*\]\(\s*data:[^)]*\)\s*`)
	inlineDataImgTag = regexp.MustCompile(`(?is)<img[^>]*\ssrc=["']?data:[^>]*>\s*`)
	fullHTMLDocument = regexp.MustCompile(`(?is)</body>|<html[\s>]`)
)

func stripInlineImages(markdown string) string {
	markdown = inlineDataImgTag.ReplaceAllString(markdown, "")
	return inlineDataImage.ReplaceAllString(markdown, "")
}

// Trafilatura's fallback extractor sometimes hands back the entire document
// instead of the article, which is not something worth storing.
func isFullDocument(markdown string) bool {
	return fullHTMLDocument.MatchString(markdown)
}

func promoteCodeBlocks(n *html.Node) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		promoteCodeBlocks(c)
		c = next
	}
	if n.Data != "code" || !strings.Contains(textOf(n), "\n") {
		return
	}
	if p := n.Parent; p == nil || p.Data == "pre" || p.Data == "code" {
		return
	}
	target := n
	if p := n.Parent; p.Data == "p" && p.Parent != nil && soleContent(p, n) {
		target = p
	}
	pre := &html.Node{Type: html.ElementNode, Data: "pre", DataAtom: atom.Pre}
	target.Parent.InsertBefore(pre, target)
	target.Parent.RemoveChild(target)
	if target != n {
		n.Parent.RemoveChild(n)
	}
	pre.AppendChild(n)
}

func soleContent(parent, child *html.Node) bool {
	return strings.TrimSpace(textOf(parent)) == strings.TrimSpace(textOf(child))
}

func textOf(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(textOf(c))
	}
	return b.String()
}
