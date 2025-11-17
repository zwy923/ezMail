package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"api-gateway/internal/model"
	"api-gateway/internal/repository"
	"mygoproject/pkg/util"
)

type Service struct {
	userRepo  *repository.UserRepository
	jwtSecret string
}

func NewService(userRepo *repository.UserRepository, jwtSecret string) *Service {
	return &Service{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

// Register creates a new user.
func (s *Service) Register(ctx context.Context, email, password string) (*model.User, error) {
	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("email already exists")
	}

	hash, err := util.HashPassword(password)
	if err != nil {
		return nil, err
	}

	u := &model.User{
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
	}

	if err := s.userRepo.CreateUser(ctx, u); err != nil {
		return nil, err
	}

	return u, nil
}

// Login checks user credentials and returns JWT.
func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return "", errors.New("invalid email or password")
	}

	if !util.CheckPassword(password, u.PasswordHash) {
		return "", errors.New("invalid email or password")
	}

	token, err := util.GenerateJWT(u.ID, s.jwtSecret)
	if err != nil {
		return "", err
	}

	return token, nil
}

