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

// Insert inserts classification metadata.
func (r *MetadataRepository) Insert(
	ctx context.Context,
	emailID int,
	category string,
	confidence float64,
) error {
	query := `
        INSERT INTO emails_metadata (email_id, category, confidence, created_at)
        VALUES ($1, $2, $3, NOW())
    `
	_, err := r.db.Exec(ctx, query, emailID, category, confidence)
	return err
}
