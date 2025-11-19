package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type HabitRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewHabitRepository(db *pgxpool.Pool, logger *zap.Logger) *HabitRepository {
	return &HabitRepository{
		db:     db,
		logger: logger,
	}
}

func (r *HabitRepository) ListAllActive(ctx context.Context) ([]HabitForToday, error) {
	query := `
        SELECT id, user_id, title, recurrence_pattern
        FROM habits
        WHERE is_active = TRUE
    `
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []HabitForToday
	for rows.Next() {
		var h HabitForToday
		if err := rows.Scan(
			&h.ID,
			&h.UserID,
			&h.Title,
			&h.RecurrencePattern,
		); err != nil {
			return nil, err
		}
		habits = append(habits, h)
	}
	return habits, nil
}

type HabitForToday struct {
	ID               int
	UserID           int
	Title            string
	RecurrencePattern string
}

