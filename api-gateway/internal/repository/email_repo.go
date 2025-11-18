package repository

import (
	"context"

	"mygoproject/contracts/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailRepository struct {
	db *pgxpool.Pool
}

func NewEmailRepository(db *pgxpool.Pool) *EmailRepository {
	return &EmailRepository{db: db}
}

func (r *EmailRepository) ListEmailsWithMetadata(ctx context.Context, userID int) ([]db.EmailWithMetadata, error) {
	query := `
        SELECT 
            r.id,
            r.subject,
            r.body,
            r.status,
            r.created_at,
            
            m.categories,
            m.priority,
            m.summary

        FROM emails_raw r
        LEFT JOIN emails_metadata m
            ON r.id = m.email_id
        
        WHERE r.user_id = $1
        ORDER BY r.created_at DESC;
    `

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []db.EmailWithMetadata

	for rows.Next() {
		var e db.EmailWithMetadata
		var categories []string
		var priority, summary *string // allow null

		err := rows.Scan(
			&e.ID,
			&e.Subject,
			&e.Body,
			&e.Status,
			&e.CreatedAt,

			&categories,
			&priority,
			&summary,
		)
		if err != nil {
			return nil, err
		}

		e.Categories = categories
		if priority != nil {
			e.Priority = *priority
		}
		if summary != nil {
			e.Summary = *summary
		}

		result = append(result, e)
	}

	return result, nil
}
