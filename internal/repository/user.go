package repository

import (
	"context"
	"database/sql"

	"github.com/dragodui/my-deploy/internal/auth"
	"github.com/dragodui/my-deploy/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.QueryRowContext(ctx, "SELECT id, email, name, password FROM users WHERE email = $1", email).Scan(&user.ID, &user.Email, &user.Name, &user.Password); err != nil {
		return nil, err
	}

	return &user, nil
}
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	if err := r.db.QueryRowContext(ctx, "SELECT id, email, name, password FROM users WHERE id = $1", id).Scan(&user.ID, &user.Email, &user.Name, &user.Password); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) Create(ctx context.Context, email, name, password string) (*models.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	var user models.User
	err = r.db.QueryRowContext(ctx, "INSERT INTO users (email, name, password) VALUES ($1, $2, $3)  RETURNING id, email, name, password", email, name, hash).Scan(&user.ID, &user.Email, &user.Name, &user.Password)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
