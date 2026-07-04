package queue

import (
	"context"
	"net/url"
	"strings"

	"github.com/redis/go-redis/v9"
)

const wwwPrefix = "www."

type Blocklist struct {
	client *redis.Client
	key    string
}

func NewBlocklist(client *redis.Client, key string) *Blocklist {
	return &Blocklist{client: client, key: key}
}

func (b *Blocklist) Load(ctx context.Context, domains []string) error {
	if err := b.client.Del(ctx, b.key).Err(); err != nil {
		return err
	}

	elems := make([]interface{}, 0, len(domains))
	for _, d := range domains {
		if d = strings.TrimPrefix(strings.ToLower(d), wwwPrefix); d != "" {
			elems = append(elems, d)
		}
	}
	if len(elems) == 0 {
		return nil
	}

	return b.client.SAdd(ctx, b.key, elems...).Err()
}

func (b *Blocklist) Blocked(ctx context.Context, u *url.URL) bool {
	if u == nil {
		return false
	}

	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), wwwPrefix)
	if host == "" {
		return false
	}

	ok, err := b.client.SIsMember(ctx, b.key, host).Result()
	return err == nil && ok
}
