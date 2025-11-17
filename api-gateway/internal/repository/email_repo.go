package repository

import (
	"context"
	"api-gateway/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailRepository struct {
	db *pgxpool.Pool
}

func NewEmailRepository(db *pgxpool.Pool) *EmailRepository {
	return &EmailRepository{db: db}
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

