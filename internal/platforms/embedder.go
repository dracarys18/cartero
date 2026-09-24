package platforms

import (
	"context"

	"cartero/internal/config"
)

type Embedder interface {
	Model() string
	Embed(ctx context.Context, inputs []string) ([][]float32, error)
}

func NewEmbedder(cfgs map[string]config.PlatformConfig) Embedder {
	for _, cfg := range cfgs {
		model := cfg.Settings.EmbeddingModel
		if !cfg.Enabled || model == "" {
			continue
		}
		switch cfg.Type {
		case "ollama":
			return NewOllamaPlatform(model)
		case "openai":
			return NewOpenAIPlatform(cfg.Settings.BaseURL, cfg.Settings.APIKey, model)
		}
	}
	return nil
}
