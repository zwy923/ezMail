package util

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Deduper struct {
	rdb    *redis.Client
	ttl    time.Duration
	logger *zap.Logger
}

func NewDeduper(rdb *redis.Client, ttl time.Duration) *Deduper {
	return &Deduper{
		rdb:    rdb,
		ttl:    ttl,
		logger: nil, // 可选，如果不需要日志可以传 nil
	}
}

// NewDeduperWithLogger creates a deduper with logger support
func NewDeduperWithLogger(rdb *redis.Client, ttl time.Duration, logger *zap.Logger) *Deduper {
	return &Deduper{
		rdb:    rdb,
		ttl:    ttl,
		logger: logger,
	}
}

// AcquireOnce tries to acquire a dedup lock for a given handler + emailID
// returns true if this is the FIRST time processing
// returns false if it's a duplicate
func (d *Deduper) AcquireOnce(ctx context.Context, handler string, emailID int) bool {
	key := fmt.Sprintf("dedup:%s:%d", handler, emailID)

	ok, err := d.rdb.SetNX(ctx, key, 1, d.ttl).Result()
	if err != nil {
		// Redis 挂了？为了安全：当 redis 不可用时，不阻止处理，返回 true
		if d.logger != nil {
			d.logger.Warn("Redis dedup check failed, allowing processing",
				zap.String("handler", handler),
				zap.Int("email_id", emailID),
				zap.Error(err),
			)
		}
		return true
	}

	// 去重命中：记录日志
	if !ok && d.logger != nil {
		d.logger.Info("Skipped duplicated event",
			zap.String("handler", handler),
			zap.Int("email_id", emailID),
			zap.String("dedup_key", key),
		)
	}

	return ok
}

