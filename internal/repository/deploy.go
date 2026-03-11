package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/dragodui/my-deploy/internal/models"
)

type DeployRepository struct {
	db *sql.DB
}

func NewDeployRepository(db *sql.DB) *DeployRepository {
	return &DeployRepository{db}
}

func (repo *DeployRepository) Create(ctx context.Context, d *models.Deployment) (*models.Deployment, error) {
	ports, _ := json.Marshal(d.Ports)
	volumes, _ := json.Marshal(d.Volumes)
	env, _ := json.Marshal(d.Env)

	var out models.Deployment
	var portsJSON, volumesJSON, envJSON []byte

	err := repo.db.QueryRowContext(ctx,
		`INSERT INTO deployments (agent_id, name, app_id, image, ports, volumes, env)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING id, agent_id, name, app_id, image, container_id, ports, volumes, env, status, created_at`,
		d.AgentID, d.Name, d.AppID, d.Image, ports, volumes, env,
	).Scan(&out.ID, &out.AgentID, &out.Name, &out.AppID, &out.Image, &out.ContainerID, &portsJSON, &volumesJSON, &envJSON, &out.Status, &out.CreatedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(portsJSON, &out.Ports)
	json.Unmarshal(volumesJSON, &out.Volumes)
	json.Unmarshal(envJSON, &out.Env)

	return &out, nil
}

func (repo *DeployRepository) GetByID(ctx context.Context, id string) (*models.Deployment, error) {
	var d models.Deployment
	var portsJSON, volumesJSON, envJSON []byte

	err := repo.db.QueryRowContext(ctx,
		`SELECT id, agent_id, name, app_id, image, container_id, ports, volumes, env, status, created_at
		 FROM deployments WHERE id = $1`, id,
	).Scan(&d.ID, &d.AgentID, &d.Name, &d.AppID, &d.Image, &d.ContainerID, &portsJSON, &volumesJSON, &envJSON, &d.Status, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(portsJSON, &d.Ports)
	json.Unmarshal(volumesJSON, &d.Volumes)
	json.Unmarshal(envJSON, &d.Env)

	return &d, nil
}

func (repo *DeployRepository) ListByAgent(ctx context.Context, agentID string) ([]models.Deployment, error) {
	rows, err := repo.db.QueryContext(ctx,
		`SELECT id, agent_id, name, app_id, image, container_id, ports, volumes, env, status, created_at
		 FROM deployments WHERE agent_id = $1`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []models.Deployment
	for rows.Next() {
		var d models.Deployment
		var portsJSON, volumesJSON, envJSON []byte

		if err := rows.Scan(&d.ID, &d.AgentID, &d.Name, &d.AppID, &d.Image, &d.ContainerID, &portsJSON, &volumesJSON, &envJSON, &d.Status, &d.CreatedAt); err != nil {
			return nil, err
		}

		json.Unmarshal(portsJSON, &d.Ports)
		json.Unmarshal(volumesJSON, &d.Volumes)
		json.Unmarshal(envJSON, &d.Env)

		deployments = append(deployments, d)
	}

	return deployments, rows.Err()
}

func (repo *DeployRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := repo.db.ExecContext(ctx, `UPDATE deployments SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (repo *DeployRepository) UpdateContainerID(ctx context.Context, id, containerID string) error {
	_, err := repo.db.ExecContext(ctx, `UPDATE deployments SET container_id = $1 WHERE id = $2`, containerID, id)
	return err
}

func (repo *DeployRepository) Delete(ctx context.Context, id string) error {
	_, err := repo.db.ExecContext(ctx, `DELETE FROM deployments WHERE id = $1`, id)
	return err
}
