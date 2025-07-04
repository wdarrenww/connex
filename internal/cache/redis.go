package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connex/internal/config"

	"github.com/go-redis/redis/v8"
)

var (
	redisClient *redis.Client
	ctx         = context.Background()
)

// Init initializes the Redis client
func Init(cfg config.RedisConfig) (*redis.Client, error) {
	var client *redis.Client

	if cfg.URL != "" {
		opt, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
		}
		client = redis.NewClient(opt)
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
			Password: cfg.Password,
			DB:       cfg.DB,
		})
	}

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	redisClient = client
	return client, nil
}

// Get returns the Redis client
func Get() *redis.Client {
	return redisClient
}

// Set stores a value in cache with TTL
func Set(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return redisClient.Set(ctx, key, data, ttl).Err()
}

// GetValue retrieves a value from cache
func GetValue(key string, dest interface{}) error {
	data, err := redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Delete removes a key from cache
func Delete(key string) error {
	return redisClient.Del(ctx, key).Err()
}

// Exists checks if a key exists
func Exists(key string) (bool, error) {
	result, err := redisClient.Exists(ctx, key).Result()
	return result > 0, err
}

// HealthCheck checks Redis connectivity
func HealthCheck() error {
	return redisClient.Ping(ctx).Err()
}
