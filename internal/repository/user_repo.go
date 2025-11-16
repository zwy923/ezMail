package repository

import (
	"context"
	"mygoproject/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser inserts a new user.
func (r *UserRepository) CreateUser(ctx context.Context, u *model.User) error {
	query := `
        INSERT INTO users (email, password_hash, created_at)
        VALUES ($1, $2, NOW())
        RETURNING id
    `
	return r.db.QueryRow(ctx, query, u.Email, u.PasswordHash).Scan(&u.ID)
}

// FindByEmail returns user by email.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
        SELECT id, email, password_hash, created_at
        FROM users
        WHERE email = $1
    `
	var u model.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
