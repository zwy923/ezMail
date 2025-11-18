package repository

import (
	"context"
	"worker-service/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MetadataRepository struct {
	db *pgxpool.Pool
}

func NewMetadataRepository(db *pgxpool.Pool) *MetadataRepository {
	return &MetadataRepository{db: db}
}

func (r *MetadataRepository) InsertDecision(
	ctx context.Context,
	emailID int,
	decision *model.AgentDecision,
) error {

	sql := `
		INSERT INTO emails_metadata
			(email_id, categories, priority, summary, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (email_id)
		DO UPDATE SET
			categories = EXCLUDED.categories,
			priority   = EXCLUDED.priority,
			summary    = EXCLUDED.summary,
			updated_at = NOW();
	`

	_, err := r.db.Exec(ctx, sql,
		emailID,
		decision.Categories,
		decision.Priority,
		decision.Summary,
	)

	return err
}

func (r *MetadataRepository) InsertUnknown(
	ctx context.Context,
	emailID int,
) error {

	sql := `
		INSERT INTO emails_metadata
			(email_id, categories, priority, summary, created_at, updated_at)
		VALUES
			($1, ARRAY['unknown'], 'LOW', 'Classified as unknown due to AI errors', NOW(), NOW())
		ON CONFLICT (email_id)
		DO UPDATE SET
			categories = ARRAY['unknown'],
			priority   = 'LOW',
			summary    = 'Classified as unknown due to AI errors',
			updated_at = NOW();
	`

	_, err := r.db.Exec(ctx, sql, emailID)
	return err
}

