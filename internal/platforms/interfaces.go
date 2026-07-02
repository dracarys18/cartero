package platforms

import "context"

type Embedder interface {
	Embed(ctx context.Context, inputs []string) ([][]float32, error)
}

type Reranker interface {
	Rerank(ctx context.Context, query string, docs []string) ([]float64, error)
}
