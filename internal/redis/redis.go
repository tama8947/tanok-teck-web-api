package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

func NewClient(ctx context.Context, redisURL string) (*goredis.Client, error) {
	opts, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("goredis.ParseURL: %w", err)
	}

	rdb := goredis.NewClient(opts)

	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return rdb, nil
}
