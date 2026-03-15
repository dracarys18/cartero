package platforms

import (
	"context"

	"github.com/ollama/ollama/api"
)

type Embedder interface {
	Embed(ctx context.Context, req *api.EmbedRequest) (*api.EmbedResponse, error)
}
