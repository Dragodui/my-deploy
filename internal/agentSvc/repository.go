package agentsvc

import (
	"context"
	"database/sql"

	"github.com/dragodui/my-deploy/internal/shared/models"
)

type AgentRepository struct {
	db *sql.DB
}

func NewAgentRepository(db *sql.DB) *AgentRepository {
	return &AgentRepository{db}
}

func (r *AgentRepository) findAgent(ctx context.Context, agent *models.Agent, query string, args ...any) error {
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&agent.ID, &agent.UserID, &agent.Token, &agent.Name, &agent.MachineID, &agent.LastSeen, &agent.CreatedAt); err != nil {
		return err
	}

	return nil
}

func (r *AgentRepository) Create(ctx context.Context, userID, token, name, machineID string) (*models.Agent, error) {
	var agent models.Agent

	if err := r.findAgent(ctx, &agent, "INSERT INTO agents (user_id, token, name, machine_id, last_seen, created_at) values ($1,$2,$3,$4,NOW(),NOW()) RETURNING id, user_id, token, name, machine_id, last_seen, created_at", userID, token, name, machineID); err != nil {
		return nil, err
	}

	return &agent, nil
}

func (r *AgentRepository) GetByID(ctx context.Context, id string) (*models.Agent, error) {
	var agent models.Agent

	err := r.findAgent(ctx, &agent, "SELECT id, user_id, token, name, machine_id, last_seen, created_at FROM agents WHERE id = $1", id)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &agent, nil
}

func (r *AgentRepository) GetByToken(ctx context.Context, token string) (*models.Agent, error) {
	var agent models.Agent

	err := r.findAgent(ctx, &agent, "SELECT id, user_id, token, name, machine_id, last_seen, created_at FROM agents WHERE token = $1 LIMIT 1", token)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &agent, nil
}

func (r *AgentRepository) GetByUserAndMachine(ctx context.Context, userID, machineID string) (*models.Agent, error) {
	var agent models.Agent

	err := r.findAgent(ctx, &agent, "SELECT id, user_id, token, name, machine_id, last_seen, created_at FROM agents WHERE user_id = $1 AND machine_id = $2 LIMIT 1", userID, machineID)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &agent, nil
}

func (r *AgentRepository) ListByUser(ctx context.Context, userID string) ([]models.Agent, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, user_id, token, name, machine_id, last_seen, created_at FROM agents WHERE user_id=$1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var a models.Agent
		if err := rows.Scan(&a.ID, &a.UserID, &a.Token, &a.Name, &a.MachineID, &a.LastSeen, &a.CreatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}

	return agents, rows.Err()
}

func (r *AgentRepository) UpdateLastSeen(ctx context.Context, agentID string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE agents SET last_seen = NOW() WHERE id = $1", agentID)
	return err
}

func (r *AgentRepository) CreateBootstrapToken(ctx context.Context, token, userID, agentName string) (*models.AgentBootstrapToken, error) {
	var bt models.AgentBootstrapToken

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO agent_bootstrap_tokens (token, user_id, agent_name, expires_at, created_at)
		VALUES ($1, $2, $3, NOW() + INTERVAL '15 minutes', NOW())
		RETURNING token, user_id, agent_name, expires_at, used_at, created_at
	`, token, userID, agentName).Scan(&bt.Token, &bt.UserID, &bt.AgentName, &bt.ExpiresAt, &bt.UsedAt, &bt.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &bt, nil
}

func (r *AgentRepository) GetBootstrapToken(ctx context.Context, token string) (*models.AgentBootstrapToken, error) {
	var bt models.AgentBootstrapToken

	err := r.db.QueryRowContext(ctx, `
		SELECT token, user_id, agent_name, expires_at, used_at, created_at
		FROM agent_bootstrap_tokens
		WHERE token = $1
		LIMIT 1
	`, token).Scan(&bt.Token, &bt.UserID, &bt.AgentName, &bt.ExpiresAt, &bt.UsedAt, &bt.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &bt, nil
}

func (r *AgentRepository) MarkBootstrapTokenUsed(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE agent_bootstrap_tokens SET used_at = NOW() WHERE token = $1 AND used_at IS NULL", token)
	return err
}
