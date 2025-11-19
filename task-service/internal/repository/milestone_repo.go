package repository

import (
	"context"

	"task-service/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type MilestoneRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewMilestoneRepository(db *pgxpool.Pool, logger *zap.Logger) *MilestoneRepository {
	return &MilestoneRepository{
		db:     db,
		logger: logger,
	}
}

func (r *MilestoneRepository) Insert(ctx context.Context, m *model.Milestone) (int, error) {
	r.logger.Debug("Inserting milestone",
		zap.Int("project_id", m.ProjectID),
		zap.String("title", m.Title),
		zap.Int("phase_order", m.PhaseOrder),
	)

	query := `
        INSERT INTO milestones (project_id, title, description, phase_order, target_date, status)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `
	var id int
	err := r.db.QueryRow(ctx, query,
		m.ProjectID,
		m.Title,
		m.Description,
		m.PhaseOrder,
		m.TargetDate,
		m.Status,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to insert milestone", zap.Error(err))
		return 0, err
	}

	r.logger.Info("Milestone inserted successfully",
		zap.Int("id", id),
		zap.Int("project_id", m.ProjectID),
	)
	return id, nil
}

func (r *MilestoneRepository) FindByProjectID(ctx context.Context, projectID int) ([]model.Milestone, error) {
	query := `
        SELECT id, project_id, title, description, phase_order, target_date, status, created_at, updated_at
        FROM milestones
        WHERE project_id = $1
        ORDER BY phase_order ASC
    `

	rows, err := r.db.Query(ctx, query, projectID)
	if err != nil {
		r.logger.Error("Failed to find milestones", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var milestones []model.Milestone
	for rows.Next() {
		var m model.Milestone
		if err := rows.Scan(
			&m.ID,
			&m.ProjectID,
			&m.Title,
			&m.Description,
			&m.PhaseOrder,
			&m.TargetDate,
			&m.Status,
			&m.CreatedAt,
			&m.UpdatedAt,
		); err != nil {
			r.logger.Error("Failed to scan milestone", zap.Error(err))
			return nil, err
		}
		milestones = append(milestones, m)
	}

	return milestones, nil
}
