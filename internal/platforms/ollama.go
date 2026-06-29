package platforms

import (
	"context"

	"github.com/ollama/ollama/api"
)

type OllamaPlatform struct {
	client *api.Client
	model  string
}

func NewOllamaPlatform(model string) *OllamaPlatform {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		panic("failed to create Ollama client: " + err.Error())
	}

	if model == "" {
		panic("failed to create Ollama client, Model cannot be empty")
	}

	return &OllamaPlatform{
		client: client,
		model:  model,
	}
}

func (o *OllamaPlatform) Client() *api.Client { return o.client }

func (o *OllamaPlatform) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	resp, err := o.Client().Embed(ctx, &api.EmbedRequest{Input: inputs, Model: o.model})
	if err != nil {
		return nil, err
	}
	return resp.Embeddings, nil
}

func (o *OllamaPlatform) Generate(ctx context.Context, request *api.GenerateRequest, respFunc api.GenerateResponseFunc) error {
	request.Model = o.model
	return o.Client().Generate(ctx, request, respFunc)
}
