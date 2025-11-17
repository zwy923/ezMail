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

// Insert inserts classification metadata with success status.
// Uses ON CONFLICT DO NOTHING for idempotency (email_id has UNIQUE constraint).
func (r *MetadataRepository) Insert(
	ctx context.Context,
	emailID int,
	category string,
	confidence float64,
) error {
	query := `
        INSERT INTO emails_metadata (email_id, category, confidence, status, created_at)
        VALUES ($1, $2, $3, 'success', NOW())
        ON CONFLICT (email_id) DO NOTHING
    `
	_, err := r.db.Exec(ctx, query, emailID, category, confidence)
	return err
}

// InsertFailed inserts a failed classification record.
// Uses ON CONFLICT DO UPDATE to allow retry attempts to update status.
func (r *MetadataRepository) InsertFailed(
	ctx context.Context,
	emailID int,
	reason string,
) error {
	query := `
        INSERT INTO emails_metadata (email_id, category, confidence, status, created_at)
        VALUES ($1, 'unknown', 0.0, $2, NOW())
        ON CONFLICT (email_id) DO UPDATE 
        SET status = $2, created_at = NOW()
    `
	_, err := r.db.Exec(ctx, query, emailID, reason)
	return err
}

