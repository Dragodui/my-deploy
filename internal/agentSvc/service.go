package agentsvc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/dragodui/my-deploy/internal/shared/models"
)

type AgentRepo interface {
	GetByID(ctx context.Context, id string) (*models.Agent, error)
	GetByUserAndMachine(ctx context.Context, userID, machineID string) (*models.Agent, error)
	GetByToken(ctx context.Context, token string) (*models.Agent, error)
	Create(ctx context.Context, userID, token, name, machineID string) (*models.Agent, error)
	ListByUser(ctx context.Context, userID string) ([]models.Agent, error)
	UpdateLastSeen(ctx context.Context, agentID string) error
	CreateBootstrapToken(ctx context.Context, token, userID, agentName string) (*models.AgentBootstrapToken, error)
	GetBootstrapToken(ctx context.Context, token string) (*models.AgentBootstrapToken, error)
	MarkBootstrapTokenUsed(ctx context.Context, token string) error
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

func (svc *AgentService) generateBootstrapToken() (string, error) {
	return svc.generateAgentToken()
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

func (svc *AgentService) GetByID(ctx context.Context, id string) (*models.Agent, error) {
	return svc.agentRepo.GetByID(ctx, id)
}

func (svc *AgentService) ListByUser(ctx context.Context, userID string) ([]models.Agent, error) {
	return svc.agentRepo.ListByUser(ctx, userID)
}

func (svc *AgentService) CreateBootstrapToken(ctx context.Context, userID, agentName string) (*models.AgentBootstrapToken, error) {
	token, err := svc.generateBootstrapToken()
	if err != nil {
		return nil, err
	}

	return svc.agentRepo.CreateBootstrapToken(ctx, token, userID, agentName)
}

func (svc *AgentService) ExchangeBootstrapToken(ctx context.Context, token, machineID string) (*models.Agent, error) {
	bt, err := svc.agentRepo.GetBootstrapToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if bt == nil {
		return nil, nil
	}
	if bt.UsedAt != nil || bt.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}

	agent, err := svc.RegisterOrGet(ctx, bt.UserID, bt.AgentName, machineID)
	if err != nil {
		return nil, err
	}

	if err := svc.agentRepo.MarkBootstrapTokenUsed(ctx, token); err != nil {
		return nil, err
	}

	return agent, nil
}
