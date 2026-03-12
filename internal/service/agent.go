package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/dragodui/my-deploy/internal/models"
)

type AgentRepo interface {
	GetByUserAndMachine(ctx context.Context, userID, machineID string) (*models.Agent, error)
	GetByToken(ctx context.Context, token string) (*models.Agent, error)
	Create(ctx context.Context, userID, token, name, machineID string) (*models.Agent, error)
	ListByUser(ctx context.Context, userID string) ([]models.Agent, error)
	UpdateLastSeen(ctx context.Context, agentID string) error
}

type AgentService struct {
	agentRepo AgentRepo
}

func NewAgentService(agentRepo AgentRepo) *AgentService {
	return &AgentService{
		agentRepo,
	}
}

func (svc *AgentService) generateAgentToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (svc *AgentService) RegisterOrGet(ctx context.Context, userID, name, machineID string) (*models.Agent, error) {
	agent, err := svc.agentRepo.GetByUserAndMachine(ctx, userID, machineID)

	if agent == nil && err == nil {
		token, err := svc.generateAgentToken()
		if err != nil {
			return nil, err
		}

		agent, err = svc.agentRepo.Create(ctx, userID, token, name, machineID)
		if err != nil {
			return nil, err
		}

		return agent, nil
	}

	return agent, err
}

func (svc *AgentService) ListByUser(ctx context.Context, userID string) ([]models.Agent, error) {
	return svc.agentRepo.ListByUser(ctx, userID)
}
