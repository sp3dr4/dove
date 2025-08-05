package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/sp3dr4/dove/internal/domain"
)

type RedisCache struct {
	client *redis.Client
	logger *slog.Logger
}

func NewRedisCache(client *redis.Client, logger *slog.Logger) *RedisCache {
	return &RedisCache{
		client: client,
		logger: logger,
	}
}

func (c *RedisCache) Get(ctx context.Context, shortCode string) (*domain.URL, error) {
	key := c.buildKey(shortCode)

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// Cache miss is not an error, just return nil
			return nil, nil
		}
		c.logger.Error("Failed to get from cache", "key", key, "error", err)
		return nil, fmt.Errorf("cache get failed: %w", err)
	}

	var url domain.URL
	if err := json.Unmarshal([]byte(val), &url); err != nil {
		c.logger.Error("Failed to unmarshal cached value", "key", key, "error", err)
		return nil, fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	return &url, nil
}

func (c *RedisCache) Set(ctx context.Context, url *domain.URL, ttl time.Duration) error {
	key := c.buildKey(url.ShortCode)

	data, err := json.Marshal(url)
	if err != nil {
		c.logger.Error("Failed to marshal URL for cache", "short_code", url.ShortCode, "error", err)
		return fmt.Errorf("failed to marshal URL: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		c.logger.Error("Failed to set cache", "key", key, "error", err)
		return fmt.Errorf("cache set failed: %w", err)
	}

	return nil
}

func (c *RedisCache) Delete(ctx context.Context, shortCode string) error {
	key := c.buildKey(shortCode)

	if err := c.client.Del(ctx, key).Err(); err != nil {
		c.logger.Error("Failed to delete from cache", "key", key, "error", err)
		return fmt.Errorf("cache delete failed: %w", err)
	}

	return nil
}

func (c *RedisCache) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		c.logger.Error("Failed to ping Redis", "error", err)
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}

func (c *RedisCache) buildKey(shortCode string) string {
	return fmt.Sprintf("url:%s", shortCode)
}
