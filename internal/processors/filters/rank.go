package filters

import (
	"context"

	"cartero/internal/platforms"
	"cartero/internal/utils/keywords"
)

type Interest struct {
	Vector  []float32
	Lexical string
}

func BuildInterests(ctx context.Context, embedder platforms.Embedder, kws []keywords.KeywordWithContext) ([]Interest, error) {
	if embedder == nil || len(kws) == 0 {
		return nil, nil
	}

	texts := make([]string, len(kws))
	labels := make([]string, len(kws))
	for i, kw := range kws {
		text := kw.Context
		if text == "" {
			text = kw.Keyword
		}
		texts[i] = text
		labels[i] = kw.Keyword
		if labels[i] == "" {
			labels[i] = kw.Context
		}
	}

	const batch = 32
	var vectors [][]float32
	for start := 0; start < len(texts); start += batch {
		end := start + batch
		if end > len(texts) {
			end = len(texts)
		}
		vecs, err := embedder.Embed(ctx, texts[start:end])
		if err != nil {
			return nil, err
		}
		vectors = append(vectors, vecs...)
	}

	out := make([]Interest, 0, len(vectors))
	for i := range vectors {
		out = append(out, Interest{Vector: vectors[i], Lexical: labels[i]})
	}
	return out, nil
}
