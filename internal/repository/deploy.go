package repository

import (
	"database/sql"

	"github.com/dragodui/my-deploy/internal/models"
)

type DeployRepository struct {
	db *sql.DB
}

func NewDeployRepository(db *sql.DB) *DeployRepository {
	return &DeployRepository{db}
}

// get service custom name + file to the yaml config
func (repo *DeployRepository) Create(deploy *models.Deployment) error {
	// mock
	return nil
}
