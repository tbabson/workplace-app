package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedis(url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := redis.NewClient(opts)

	if err := pingWithRetry("redis", func(ctx context.Context) error { return client.Ping(ctx).Err() }); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
