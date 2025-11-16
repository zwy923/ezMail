package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetadataRepository struct {
	db *pgxpool.Pool
}

func NewMetadataRepository(db *pgxpool.Pool) *MetadataRepository {
	return &MetadataRepository{db: db}
}

// Exists checks if metadata already exists for an email.
func (r *MetadataRepository) Exists(ctx context.Context, emailID int) (bool, error) {
	query := `
        SELECT EXISTS(
            SELECT 1 FROM emails_metadata WHERE email_id = $1
        )
    `
	var exists bool
	err := r.db.QueryRow(ctx, query, emailID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// Insert inserts classification metadata.
// Returns nil if metadata already exists (idempotent).
func (r *MetadataRepository) Insert(
	ctx context.Context,
	emailID int,
	category string,
	confidence float64,
) error {
	// 使用 INSERT ... ON CONFLICT DO NOTHING 实现幂等性
	// 但需要先添加唯一约束，或者先检查是否存在
	exists, err := r.Exists(ctx, emailID)
	if err != nil {
		return err
	}
	if exists {
		// 已存在，幂等返回
		return nil
	}

	query := `
        INSERT INTO emails_metadata (email_id, category, confidence, created_at)
        VALUES ($1, $2, $3, NOW())
    `
	_, err = r.db.Exec(ctx, query, emailID, category, confidence)
	if err != nil {
		// 如果是唯一约束冲突，也认为是幂等的
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil
		}
		return err
	}
	return nil
}
