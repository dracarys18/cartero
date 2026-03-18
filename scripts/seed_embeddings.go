//go:build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"cartero/internal/config"
	"cartero/internal/platforms"
	"cartero/internal/queue"
	"cartero/internal/utils/file"

	ollamaapi "github.com/ollama/ollama/api"
	"github.com/redis/go-redis/v9"
)

type entry struct {
	Keyword string `json:"keyword"`
	Context string `json:"context_string"`
}

func main() {
	ctx := context.Background()

	cfg, err := config.Load("config-seed.toml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	data, err := loadKeywordFiles(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read keywords: %v\n", err)
		os.Exit(1)
	}

	var entries []entry
	if err := json.Unmarshal(data, &entries); err != nil {
		fmt.Fprintf(os.Stderr, "parse keywords: %v\n", err)
		os.Exit(1)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:          cfg.Redis.Addr,
		Password:      cfg.Redis.Password,
		DB:            cfg.Redis.DB,
		UnstableResp3: true,
		Protocol:      3,
	})
	defer redisClient.Close()

	embedCache := queue.NewEmbedCache(redisClient)

	ollamaCfg, ok := cfg.Platforms["ollama"]
	if !ok || !ollamaCfg.Enabled {
		fmt.Fprintln(os.Stderr, "ollama platform not configured or disabled")
		os.Exit(1)
	}

	embedder := platforms.NewOllamaPlatform(ollamaCfg.Settings.EmbeddingModel)

	var missing []entry
	for _, e := range entries {
		_, found, err := embedCache.Get(ctx, e.Keyword)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cache get %q: %v\n", e.Keyword, err)
			os.Exit(1)
		}
		if !found {
			missing = append(missing, e)
		}
	}

	fmt.Printf("total: %d  cached: %d  missing: %d\n", len(entries), len(entries)-len(missing), len(missing))

	if len(missing) == 0 {
		fmt.Println("nothing to seed")
		return
	}

	contexts := make([]string, len(missing))
	for i, e := range missing {
		contexts[i] = e.Context
	}

	resp, err := embedder.Embed(ctx, &ollamaapi.EmbedRequest{Input: contexts})
	if err != nil {
		fmt.Fprintf(os.Stderr, "embed: %v\n", err)
		os.Exit(1)
	}

	if len(resp.Embeddings) != len(missing) {
		fmt.Fprintf(os.Stderr, "embedding count mismatch: got %d want %d\n", len(resp.Embeddings), len(missing))
		os.Exit(1)
	}

	dims := len(resp.Embeddings[0])

	for i, e := range missing {
		vec := resp.Embeddings[i]
		if err := embedCache.Set(ctx, e.Keyword, vec); err != nil {
			fmt.Fprintf(os.Stderr, "cache set %q: %v\n", e.Keyword, err)
			os.Exit(1)
		}
	}

	if err := embedCache.SetDims(ctx, dims); err != nil {
		fmt.Fprintf(os.Stderr, "set dims: %v\n", err)
		os.Exit(1)
	}

	if err := embedCache.EnsureIndex(ctx, dims); err != nil {
		fmt.Fprintf(os.Stderr, "ensure index: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("stored: %d  dims: %d  HNSW index ready\n", len(missing), dims)
	fmt.Println("done")
}

func loadKeywordFiles(config *config.Config) ([]byte, error) {
	for name, proc := range config.Processors {
		if proc.Settings.KeywordsFile == "" {
			continue
		}

		file := file.NewFile(proc.Settings.KeywordsFile)
		data, err := file.Get()
		if err != nil {
			return nil, fmt.Errorf("processor %q: %w", name, err)
		} else {
			return data, nil
		}

	}
	return nil, nil
}
