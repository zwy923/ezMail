package redis

import (
    "context"
    "github.com/redis/go-redis/v9"
    "mygoproject/config"
)

var Rdb *redis.Client
var Ctx = context.Background()

func NewRedisClient(cfg config.RedisConfig) *redis.Client {
    Rdb = redis.NewClient(&redis.Options{
        Addr:     cfg.Addr,
        Password: cfg.Password,
        DB:       cfg.DB,
    })
    return Rdb
}
