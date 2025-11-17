package util

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

type Deduper struct {
    rdb *redis.Client
    ttl time.Duration
}

func NewDeduper(rdb *redis.Client, ttl time.Duration) *Deduper {
    return &Deduper{rdb: rdb, ttl: ttl}
}

// AcquireOnce tries to acquire a dedup lock for a given handler + emailID
// returns true if this is the FIRST time processing
// returns false if it's a duplicate
func (d *Deduper) AcquireOnce(ctx context.Context, handler string, emailID int) bool {
    key := fmt.Sprintf("dedup:%s:%d", handler, emailID)

    ok, err := d.rdb.SetNX(ctx, key, 1, d.ttl).Result()
    if err != nil {
        // Redis 挂了？为了安全：当 redis 不可用时，不阻止处理，返回 true
        return true
    }
    return ok
}