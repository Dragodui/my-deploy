package repository

import "database/sql"

type DeployRepository struct {
	db *sql.DB
}

func NewDeployRepository(db *sql.DB) *DeployRepository {
	return &DeployRepository{db}
}

// get service custom name + file to the yaml config
func (repo *DeployRepository) Create(name, file string) error {
	// mock
	return nil
}