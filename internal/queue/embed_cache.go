package queue

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const (
	embedKeyPrefix   = ":embed:kw:"
	embedDimsKey     = ":embed:dims"
	embedIndexSuffix = ":embed:kw:idx"
	embeddingField   = "embedding"
)

type KNNResult struct {
	Keyword string
	Score   float64
}

type EmbedCache struct {
	client    *redis.Client
	prefix    string
	indexName string
}

func NewEmbedCache(client *redis.Client) *EmbedCache {
	prefix := os.Getenv("CARTERO_QUEUE_PREFIX")
	if prefix == "" {
		prefix = defaultPrefix
	}
	return &EmbedCache{
		client:    client,
		prefix:    prefix,
		indexName: prefix + embedIndexSuffix,
	}
}

func (e *EmbedCache) kwKey(keyword string) string {
	return e.prefix + embedKeyPrefix + keyword
}

func (e *EmbedCache) dimsKey() string {
	return e.prefix + embedDimsKey
}

func encodeVec(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

func decodeVec(buf []byte) ([]float32, error) {
	if len(buf)%4 != 0 {
		return nil, fmt.Errorf("invalid embedding bytes length %d", len(buf))
	}
	vec := make([]float32, len(buf)/4)
	for i := range vec {
		vec[i] = math.Float32frombits(binary.LittleEndian.Uint32(buf[i*4:]))
	}
	return vec, nil
}

func (e *EmbedCache) Set(ctx context.Context, keyword string, vec []float32) error {
	return e.client.HSet(ctx, e.kwKey(keyword), embeddingField, encodeVec(vec)).Err()
}

func (e *EmbedCache) Get(ctx context.Context, keyword string) ([]float32, bool, error) {
	val, err := e.client.HGet(ctx, e.kwKey(keyword), embeddingField).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	vec, err := decodeVec(val)
	if err != nil {
		return nil, false, err
	}
	return vec, true, nil
}

func (e *EmbedCache) GetDims(ctx context.Context) (int, error) {
	val, err := e.client.Get(ctx, e.dimsKey()).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(val)
}

func (e *EmbedCache) SetDims(ctx context.Context, dims int) error {
	return e.client.Set(ctx, e.dimsKey(), dims, 0).Err()
}

func (e *EmbedCache) EnsureIndex(ctx context.Context, dims int) error {
	_, err := e.client.FTInfo(ctx, e.indexName).Result()
	if err == nil {
		return nil
	}
	if !errors.Is(err, redis.Nil) && err.Error() != "Unknown index name" {
		return fmt.Errorf("FTInfo: %w", err)
	}

	return e.client.FTCreate(ctx, e.indexName,
		&redis.FTCreateOptions{
			OnHash: true,
			Prefix: []interface{}{e.prefix + embedKeyPrefix},
		},
		&redis.FieldSchema{
			FieldName: embeddingField,
			FieldType: redis.SearchFieldTypeVector,
			VectorArgs: &redis.FTVectorArgs{
				HNSWOptions: &redis.FTHNSWOptions{
					Type:           "FLOAT32",
					Dim:            dims,
					DistanceMetric: "COSINE",
				},
			},
		},
	).Err()
}

func (e *EmbedCache) KNNSearch(ctx context.Context, k int, queryVec []float32) ([]KNNResult, error) {
	scoreField := fmt.Sprintf("__%s_score", embeddingField)
	query := fmt.Sprintf("(*)=>[KNN %d @%s $vec AS %s]", k, embeddingField, scoreField)
	raw, err := e.client.Do(ctx,
		"FT.SEARCH", e.indexName, query,
		"PARAMS", "2", "vec", encodeVec(queryVec),
		"RETURN", "1", scoreField,
		"SORTBY", scoreField, "ASC",
		"DIALECT", "2",
	).Result()
	if err != nil {
		return nil, fmt.Errorf("KNNSearch: %w", err)
	}

	resp, ok := raw.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("KNNSearch: unexpected result type %T", raw)
	}

	docs, _ := resp["results"].([]interface{})
	kwPrefix := e.prefix + embedKeyPrefix
	results := make([]KNNResult, 0, len(docs))
	for _, d := range docs {
		doc, ok := d.(map[interface{}]interface{})
		if !ok {
			continue
		}
		id, _ := doc["id"].(string)
		extras, _ := doc["extra_attributes"].(map[interface{}]interface{})
		distStr, _ := extras[scoreField].(string)
		dist, err := strconv.ParseFloat(distStr, 64)
		if err != nil {
			continue
		}
		results = append(results, KNNResult{
			Keyword: id[len(kwPrefix):],
			Score:   1 - dist,
		})
	}
	return results, nil
}
