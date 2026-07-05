package queue

import (
	"bytes"
	"context"
	"encoding/binary"
	"time"

	"github.com/redis/go-redis/v9"
)

type EmbedCache struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

func NewEmbedCache(client *redis.Client, prefix string, ttl time.Duration) *EmbedCache {
	return &EmbedCache{client: client, prefix: prefix, ttl: ttl}
}

func (c *EmbedCache) key(hash string) string {
	return c.prefix + ":embcache:" + hash
}

func (c *EmbedCache) Get(ctx context.Context, hash string) [][]float32 {
	b, err := c.client.Get(ctx, c.key(hash)).Bytes()
	if err != nil {
		return nil
	}
	return decodeEmbedding(b)
}

func (c *EmbedCache) Set(ctx context.Context, hash string, embedding [][]float32) {
	_ = c.client.Set(ctx, c.key(hash), encodeEmbedding(embedding), c.ttl).Err()
}

func encodeEmbedding(emb [][]float32) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(emb)))
	for _, v := range emb {
		_ = binary.Write(buf, binary.LittleEndian, uint32(len(v)))
		_ = binary.Write(buf, binary.LittleEndian, v)
	}
	return buf.Bytes()
}

func decodeEmbedding(b []byte) [][]float32 {
	r := bytes.NewReader(b)
	var n uint32
	if binary.Read(r, binary.LittleEndian, &n) != nil {
		return nil
	}
	out := make([][]float32, n)
	for i := range out {
		var l uint32
		if binary.Read(r, binary.LittleEndian, &l) != nil {
			return nil
		}
		v := make([]float32, l)
		if binary.Read(r, binary.LittleEndian, v) != nil {
			return nil
		}
		out[i] = v
	}
	return out
}
