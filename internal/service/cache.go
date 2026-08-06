package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheService struct {
	RedisClient *redis.Client
}

func (c *CacheService) Get(key string) (string, error) {
	ctx := context.Background()
	val, err := c.RedisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("CacheService.Get: %w", err)
	}
	return val, nil
}

func (c *CacheService) Set(key, value string, ttl time.Duration) error {
	ctx := context.Background()
	err := c.RedisClient.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("CacheService.Set: %w", err)
	}
	return nil
}

func (c *CacheService) Delete(key string) error {
	ctx := context.Background()
	err := c.RedisClient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("CacheService.Delete: %w", err)
	}
	return nil
}

func (c *CacheService) Remember(key string, ttl time.Duration, fn func() (string, error)) (string, error) {
	val, err := c.Get(key)
	if err != nil {
		return "", err
	}
	if val != "" {
		return val, nil
	}

	val, err = fn()
	if err != nil {
		return "", fmt.Errorf("CacheService.Remember fn: %w", err)
	}

	if setErr := c.Set(key, val, ttl); setErr != nil {
		return "", setErr
	}

	return val, nil
}
