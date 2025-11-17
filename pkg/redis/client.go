package redis

import (
	"context"

	"mygoproject/pkg/config"

	"github.com/redis/go-redis/v9"
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

