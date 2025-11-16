package repository

import (
	"context"
	"mygoproject/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailRepository struct {
	db *pgxpool.Pool
}

func NewEmailRepository(db *pgxpool.Pool) *EmailRepository {
	return &EmailRepository{db: db}
}

// CreateRawEmail inserts the raw email.
func (r *EmailRepository) CreateRawEmail(ctx context.Context, e *model.EmailRaw) (int, error) {
	query := `
        INSERT INTO emails_raw (user_id, subject, body, raw_json, status, created_at)
        VALUES ($1, $2, $3, $4, 'received', NOW())
        RETURNING id
    `
	var id int
	err := r.db.QueryRow(ctx, query, e.UserID, e.Subject, e.Body, e.RawJSON).Scan(&id)
	return id, err
}

// FindRawByID returns raw email by id.
func (r *EmailRepository) FindRawByID(ctx context.Context, id int) (*model.EmailRaw, error) {
	query := `
        SELECT id, user_id, subject, body, raw_json, status, created_at
        FROM emails_raw
        WHERE id = $1
    `
	var e model.EmailRaw
	err := r.db.QueryRow(ctx, query, id).Scan(
		&e.ID,
		&e.UserID,
		&e.Subject,
		&e.Body,
		&e.RawJSON,
		&e.Status,
		&e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
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

// ListEmailsWithMetadata returns all emails + metadata for a user.
func (r *EmailRepository) ListEmailsWithMetadata(ctx context.Context, userID int) ([]model.EmailWithMetadata, error) {
	query := `
        SELECT 
            r.id,
            r.subject,
            r.body,
            r.status,
            r.created_at,
            m.category,
            m.confidence
        FROM emails_raw r
        LEFT JOIN emails_metadata m
        ON r.id = m.email_id
        WHERE r.user_id = $1
        ORDER BY r.created_at DESC
    `

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emails := []model.EmailWithMetadata{}

	for rows.Next() {
		var e model.EmailWithMetadata
		var metadataCategory *string
		var metadataConfidence *float64

		err := rows.Scan(
			&e.ID,
			&e.Subject,
			&e.Body,
			&e.Status,
			&e.CreatedAt,
			&metadataCategory,
			&metadataConfidence,
		)
		if err != nil {
			return nil, err
		}

		if metadataCategory != nil && metadataConfidence != nil {
			e.Metadata = &model.EmailMetadata{
				Category:   *metadataCategory,
				Confidence: *metadataConfidence,
			}
		}

		emails = append(emails, e)
	}

	return emails, nil
}
