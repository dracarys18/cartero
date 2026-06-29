package platforms

import "context"

type Embedder interface {
	Embed(ctx context.Context, inputs []string) ([][]float32, error)
}
