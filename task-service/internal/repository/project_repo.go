package repository

import (
	"context"

	"task-service/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type ProjectRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewProjectRepository(db *pgxpool.Pool, logger *zap.Logger) *ProjectRepository {
	return &ProjectRepository{
		db:     db,
		logger: logger,
	}
}

func (r *ProjectRepository) Insert(ctx context.Context, p *model.Project) (int, error) {
	r.logger.Debug("Inserting project",
		zap.Int("user_id", p.UserID),
		zap.String("title", p.Title),
	)

	query := `
        INSERT INTO projects (user_id, title, description, target_date, status)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `
	var id int
	err := r.db.QueryRow(ctx, query,
		p.UserID,
		p.Title,
		p.Description,
		p.TargetDate,
		p.Status,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to insert project", zap.Error(err))
		return 0, err
	}

	r.logger.Info("Project inserted successfully",
		zap.Int("id", id),
		zap.Int("user_id", p.UserID),
	)
	return id, nil
}
