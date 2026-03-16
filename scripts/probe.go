//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"cartero/internal/config"
	"cartero/internal/platforms"
	"cartero/internal/queue"

	ollamaapi "github.com/ollama/ollama/api"
	"github.com/redis/go-redis/v9"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/viterin/vek/vek32"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: probe <text>")
		os.Exit(1)
	}
	text := os.Args[1]

	ctx := context.Background()

	cfg, err := config.Load("config.toml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	ollamaCfg, ok := cfg.Platforms["ollama"]
	if !ok || !ollamaCfg.Enabled {
		fmt.Fprintln(os.Stderr, "ollama platform not configured or disabled")
		os.Exit(1)
	}

	embedder := platforms.NewOllamaPlatform(ollamaCfg.Settings.EmbeddingModel)

	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(1800),
		textsplitter.WithChunkOverlap(1800/8),
	)

	chunks, err := splitter.SplitText(text)
	if err != nil || len(chunks) == 0 {
		fmt.Fprintln(os.Stderr, "failed to split text")
		os.Exit(1)
	}

	resp, err := embedder.Embed(ctx, &ollamaapi.EmbedRequest{Input: chunks})
	if err != nil {
		fmt.Fprintf(os.Stderr, "embed: %v\n", err)
		os.Exit(1)
	}

	avg := make([]float32, len(resp.Embeddings[0]))
	for _, v := range resp.Embeddings {
		vek32.Add_Inplace(avg, v)
	}
	vek32.DivNumber_Inplace(avg, float32(len(resp.Embeddings)))

	redisClient := redis.NewClient(&redis.Options{
		Addr:          cfg.Redis.Addr,
		Password:      cfg.Redis.Password,
		DB:            cfg.Redis.DB,
		UnstableResp3: true,
	})
	defer redisClient.Close()

	embedCache := queue.NewEmbedCache(redisClient)

	results, err := embedCache.KNNSearch(ctx, 5, avg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "KNNSearch: %v\n", err)
		os.Exit(1)
	}

	for i, r := range results {
		fmt.Printf("%d. %-40s %.4f\n", i+1, r.Keyword, r.Score)
	}
}
