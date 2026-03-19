package authsvc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dragodui/my-deploy/internal/shared/auth"
	"github.com/dragodui/my-deploy/internal/shared/models"
)

type UserRepo interface {
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	Create(ctx context.Context, email, name, password string) (*models.User, error)
}

type AuthService struct {
	userRepo  UserRepo
	jwtSecret string
}

func NewAuthService(userRepo UserRepo, jwtSecret string) *AuthService {
	return &AuthService{userRepo, jwtSecret}
}

func (svc *AuthService) SignUp(ctx context.Context, email, name, password string) (string, string, error) {
	exists, err := svc.userRepo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", "", err
	}

	if exists != nil {
		return "", "", errors.New("user already exists")
	}

	user, err := svc.userRepo.Create(ctx, email, name, password)
	if err != nil {
		return "", "", err
	}

	token, err := auth.GenerateToken(user.ID, svc.jwtSecret)
	if err != nil {
		return "", "", err
	}

	return token, user.Name, nil
}

func (svc *AuthService) Me(ctx context.Context, userID string) (*models.User, error) {
	return svc.userRepo.GetByID(ctx, userID)
}

func (svc *AuthService) SignIn(ctx context.Context, email, password string) (string, string, error) {
	user, err := svc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", err
	}

	if user == nil {
		return "", "", errors.New("invalid credentials")
	}

	if !auth.CheckPassword(password, user.Password) {
		return "", "", errors.New("invalid credentials")
	}

	token, err := auth.GenerateToken(user.ID, svc.jwtSecret)
	if err != nil {
		return "", "", err
	}

	return token, user.Name, nil
}
