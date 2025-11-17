package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"mygoproject/pkg/config"
)

func NewConnection(cfg config.DBConfig, logger *zap.Logger) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
	)

	logger.Info("Initializing PostgreSQL connection pool",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("db", cfg.Name),
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Error("Failed to parse db config", zap.Error(err))
		return nil, fmt.Errorf("failed to parse db config: %w", err)
	}

	poolCfg.MaxConns = 10
	poolCfg.MinConns = 2
	poolCfg.MaxConnIdleTime = time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbpool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		logger.Fatal("PostgreSQL connection failed", zap.Error(err))
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer pingCancel()

	if err := dbpool.Ping(pingCtx); err != nil {
		logger.Fatal("PostgreSQL ping failed", zap.Error(err))
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	logger.Info("PostgreSQL connection established successfully")
	return dbpool, nil
}

