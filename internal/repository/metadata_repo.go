package repository

import (
	"context"

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
// Uses ON CONFLICT DO NOTHING for idempotency (email_id has UNIQUE constraint).
func (r *MetadataRepository) Insert(
	ctx context.Context,
	emailID int,
	category string,
	confidence float64,
) error {
	query := `
        INSERT INTO emails_metadata (email_id, category, confidence, created_at)
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (email_id) DO NOTHING
    `
	_, err := r.db.Exec(ctx, query, emailID, category, confidence)
	return err
}
