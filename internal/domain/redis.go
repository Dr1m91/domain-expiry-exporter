package domain

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RedisBackend stores entries as JSON values of a single Redis hash.
type RedisBackend struct {
	client *redis.Client
	key    string
}

func NewRedisBackend(client *redis.Client, key string) *RedisBackend {
	return &RedisBackend{client: client, key: key}
}

func (b *RedisBackend) LoadAll(ctx context.Context) ([]Entry, error) {
	values, err := b.client.HGetAll(ctx, b.key).Result()
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(values))
	for d, raw := range values {
		var e Entry
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			log.Warn().Err(err).Msgf("skipping unreadable entry for %s", d)
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (b *RedisBackend) Save(ctx context.Context, entries []Entry) error {
	values := make(map[string]interface{}, len(entries))
	for _, e := range entries {
		raw, err := json.Marshal(e)
		if err != nil {
			return err
		}
		values[e.Domain] = raw
	}
	return b.client.HSet(ctx, b.key, values).Err()
}

func (b *RedisBackend) Remove(ctx context.Context, domains []string) error {
	return b.client.HDel(ctx, b.key, domains...).Err()
}
