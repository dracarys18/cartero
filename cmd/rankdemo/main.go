package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"cartero/internal/config"
	"cartero/internal/platforms"
	"cartero/internal/processors/filters"
	"cartero/internal/storage"
	_ "cartero/internal/storage/postgres"
)

func main() {
	configPath := flag.String("config", "config.toml", "Path to configuration file")
	interestArg := flag.String("interests", "", "Interests, one per line or separated by ';'")
	limit := flag.Int("limit", 20, "Number of results")
	wSem := flag.Float64("w-semantic", 0.7, "Semantic weight")
	wLex := flag.Float64("w-lexical", 0.3, "Lexical weight")
	semFloor := flag.Float64("semantic-floor", 0.45, "Semantic baseline; similarity at/below this maps to 0")
	minScore := flag.Float64("min-score", 0, "Minimum combined score")
	lambda := flag.Float64("mmr-lambda", 0.7, "MMR diversity lambda (0<λ<1)")
	lookback := flag.Duration("lookback", 0, "Only rank items newer than this (0 = no limit)")
	flag.Parse()

	ctx := context.Background()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if cfg.Storage.Type != "postgres" {
		log.Fatalf("personalized ranking requires the postgres storage backend (got %q)", cfg.Storage.Type)
	}

	st, err := storage.New(ctx, cfg.Storage)
	if err != nil {
		log.Fatalf("open storage: %v", err)
	}
	defer func() { _ = st.Close(ctx) }()

	embedder := buildEmbedder(cfg)
	if embedder == nil {
		log.Fatalf("no enabled embedding platform found in config")
	}

	pairs := interestsFromArg(*interestArg)
	if len(pairs) == 0 {
		pairs = interestsFromConfig(cfg)
	}
	if len(pairs) == 0 {
		log.Fatalf("provide -interests, or set [interests].keywords_file in the config")
	}

	embedTexts := make([]string, len(pairs))
	for i, p := range pairs {
		embedTexts[i] = p.embed
	}

	vecs, err := embedder.Embed(ctx, embedTexts)
	if err != nil {
		log.Fatalf("embed interests: %v", err)
	}

	interests := make([]filters.Interest, 0, len(pairs))
	for i, p := range pairs {
		if i >= len(vecs) {
			break
		}
		interests = append(interests, filters.Interest{Vector: vecs[i], Lexical: p.lexical})
	}

	r := filters.NewRanker(st.Entries())
	results, err := r.Rank(ctx, interests, filters.Options{
		WSemantic:     *wSem,
		WLexical:      *wLex,
		SemanticFloor: *semFloor,
		MinScore:      *minScore,
		MMRLambda:     *lambda,
		Limit:         *limit,
		Lookback:      *lookback,
	})
	if err != nil {
		log.Fatalf("rank: %v", err)
	}

	fmt.Printf("%d interests loaded\n\n", len(pairs))
	fmt.Printf("%-4s %-6s %-6s %-6s  %s\n", "#", "score", "sem", "lex", "title")
	for i, res := range results {
		fmt.Printf("%-4d %-6.3f %-6.3f %-6.3f  %s\n", i+1, res.Score, res.Semantic, res.Lexical, res.Entry.Title)
	}
}

type interestPair struct {
	embed   string
	lexical string
}

func interestsFromArg(arg string) []interestPair {
	raw := strings.FieldsFunc(arg, func(r rune) bool { return r == ';' || r == '\n' })
	var out []interestPair
	for _, s := range raw {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, interestPair{embed: s, lexical: s})
		}
	}
	return out
}

func interestsFromConfig(cfg *config.Config) []interestPair {
	var out []interestPair
	for _, kw := range cfg.Interests.Keywords {
		embed := kw.Context
		if embed == "" {
			embed = kw.Keyword
		}
		if embed == "" {
			continue
		}
		lexical := kw.Keyword
		if lexical == "" {
			lexical = kw.Context
		}
		out = append(out, interestPair{embed: embed, lexical: lexical})
	}
	return out
}

func buildEmbedder(cfg *config.Config) platforms.Embedder {
	for _, pc := range cfg.Platforms {
		if !pc.Enabled {
			continue
		}
		model := pc.Settings.OllamaPlatformSettings.EmbeddingModel
		if model == "" {
			continue
		}
		switch pc.Type {
		case "ollama":
			return platforms.NewOllamaPlatform(model)
		case "openai":
			return platforms.NewOpenAIPlatform(
				pc.Settings.OpenAIPlatformSettings.BaseURL,
				pc.Settings.OpenAIPlatformSettings.APIKey,
				model,
			)
		}
	}
	return nil
}
