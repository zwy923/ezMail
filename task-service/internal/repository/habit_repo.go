package repository

import (
	"context"
	"task-service/internal/model"

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

func (r *HabitRepository) Insert(ctx context.Context, h *model.Habit) (int, error) {
	r.logger.Debug("Inserting habit",
		zap.Int("user_id", h.UserID),
		zap.String("title", h.Title),
		zap.String("recurrence_pattern", h.RecurrencePattern),
	)

	query := `
        INSERT INTO habits (user_id, title, recurrence_pattern, is_active)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `
	var id int
	err := r.db.QueryRow(ctx, query,
		h.UserID,
		h.Title,
		h.RecurrencePattern,
		h.IsActive,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to insert habit", zap.Error(err))
		return 0, err
	}

	r.logger.Info("Habit inserted successfully",
		zap.Int("id", id),
		zap.Int("user_id", h.UserID),
	)
	return id, nil
}

func (r *HabitRepository) ListActiveByUser(ctx context.Context, userID int) ([]model.Habit, error) {
	r.logger.Debug("Listing active habits for user", zap.Int("user_id", userID))

	query := `
        SELECT id, user_id, title, recurrence_pattern, is_active, created_at, updated_at
        FROM habits
        WHERE user_id = $1 AND is_active = TRUE
        ORDER BY created_at DESC
    `

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.logger.Error("Failed to list habits", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var habits []model.Habit
	for rows.Next() {
		var h model.Habit
		if err := rows.Scan(
			&h.ID,
			&h.UserID,
			&h.Title,
			&h.RecurrencePattern,
			&h.IsActive,
			&h.CreatedAt,
			&h.UpdatedAt,
		); err != nil {
			r.logger.Error("Failed to scan habit", zap.Error(err))
			return nil, err
		}
		habits = append(habits, h)
	}

	r.logger.Debug("Listed habits",
		zap.Int("user_id", userID),
		zap.Int("count", len(habits)),
	)
	return habits, nil
}

func (r *HabitRepository) ListAllActive(ctx context.Context) ([]model.Habit, error) {
	r.logger.Debug("Listing all active habits")

	query := `
        SELECT id, user_id, title, recurrence_pattern, is_active, created_at, updated_at
        FROM habits
        WHERE is_active = TRUE
        ORDER BY created_at DESC
    `

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.logger.Error("Failed to list all active habits", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var habits []model.Habit
	for rows.Next() {
		var h model.Habit
		if err := rows.Scan(
			&h.ID,
			&h.UserID,
			&h.Title,
			&h.RecurrencePattern,
			&h.IsActive,
			&h.CreatedAt,
			&h.UpdatedAt,
		); err != nil {
			r.logger.Error("Failed to scan habit", zap.Error(err))
			return nil, err
		}
		habits = append(habits, h)
	}

	r.logger.Debug("Listed all active habits", zap.Int("count", len(habits)))
	return habits, nil
}

