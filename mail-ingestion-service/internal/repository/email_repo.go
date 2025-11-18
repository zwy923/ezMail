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

// CreateRawEmail inserts the raw email.
func (r *EmailRepository) CreateRawEmail(ctx context.Context, e *db.Email) (int, error) {
	query := `
        INSERT INTO emails_raw (user_id, subject, body, raw_json, status, created_at)
        VALUES ($1, $2, $3, $4, 'received', NOW())
        RETURNING id
    `
	var id int
	err := r.db.QueryRow(ctx, query, e.UserID, e.Subject, e.Body, e.RawJSON).Scan(&id)
	return id, err
}

