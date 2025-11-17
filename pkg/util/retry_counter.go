package util

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RetryCounter struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRetryCounter(rdb *redis.Client, ttl time.Duration) *RetryCounter {
	return &RetryCounter{rdb: rdb, ttl: ttl}
}

// IncrementAndGet increments the retry count for a given key and returns the new count
func (r *RetryCounter) IncrementAndGet(ctx context.Context, key string) (int64, error) {
	count, err := r.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// Set expiration on first increment
	if count == 1 {
		r.rdb.Expire(ctx, key, r.ttl)
	}

	return count, nil
}

// Get returns the current retry count
func (r *RetryCounter) Get(ctx context.Context, key string) (int64, error) {
	count, err := r.rdb.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

// Reset resets the retry count
func (r *RetryCounter) Reset(ctx context.Context, key string) error {
	return r.rdb.Del(ctx, key).Err()
}

// FormatKey formats a retry key for a handler and emailID
func FormatRetryKey(handler string, emailID int) string {
	return fmt.Sprintf("retry:%s:%d", handler, emailID)
}

