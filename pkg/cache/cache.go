package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Cache { return &Cache{rdb: rdb} }

// Get retrieves and unmarshals a cached value. Returns (value, true) on hit.
func Get[T any](ctx context.Context, c *Cache, key string) (*T, bool) {
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, false
	}
	return &v, true
}

// Set marshals and stores a value with a TTL. Errors are silently dropped
// so a cache write failure never breaks the request.
func Set(ctx context.Context, c *Cache, key string, v any, ttl time.Duration) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	c.rdb.Set(ctx, key, data, ttl)
}

// Delete removes one or more keys from the cache.
func Delete(ctx context.Context, c *Cache, keys ...string) {
	if len(keys) > 0 {
		c.rdb.Del(ctx, keys...)
	}
}
