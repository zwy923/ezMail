package repository

import (
	"context"
	"worker-service/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailRepository struct {
	db *pgxpool.Pool
}

func NewEmailRepository(db *pgxpool.Pool) *EmailRepository {
	return &EmailRepository{db: db}
}

// FindRawWithMetadataByID returns raw email with metadata in a single query.
// Returns the email and whether metadata exists.
func (r *EmailRepository) FindRawWithMetadataByID(ctx context.Context, id int) (*model.EmailRaw, bool, error) {
	query := `
        SELECT 
            r.id,
            r.user_id,
            r.subject,
            r.body,
            r.raw_json,
            r.status,
            r.created_at,
            m.id as metadata_id
        FROM emails_raw r
        LEFT JOIN emails_metadata m ON r.id = m.email_id
        WHERE r.id = $1
    `
	var e model.EmailRaw
	var metadataID *int
	err := r.db.QueryRow(ctx, query, id).Scan(
		&e.ID,
		&e.UserID,
		&e.Subject,
		&e.Body,
		&e.RawJSON,
		&e.Status,
		&e.CreatedAt,
		&metadataID,
	)
	if err != nil {
		return nil, false, err
	}
	return &e, metadataID != nil, nil
}

// UpdateStatus sets raw email status (e.g. classified).
func (r *EmailRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	query := `
        UPDATE emails_raw
        SET status = $1
        WHERE id = $2
    `
	_, err := r.db.Exec(ctx, query, status, id)
	return err
}

