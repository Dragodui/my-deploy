package agentsvc

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/dragodui/my-deploy/internal/shared/models"
)

// mock repo
type mockAgentRepo struct {
	getByIDFn             func(ctx context.Context, id string) (*models.Agent, error)
	getByUserAndMachineFn func(ctx context.Context, userID, machineID string) (*models.Agent, error)
	getByTokenFn          func(ctx context.Context, token string) (*models.Agent, error)
	createFn              func(ctx context.Context, userID, token, name, machineID string) (*models.Agent, error)
	listByUserFn          func(ctx context.Context, userID string) ([]models.Agent, error)
	updateLastSeenFn      func(ctx context.Context, agentID string) error
}

func (m *mockAgentRepo) GetByID(ctx context.Context, id string) (*models.Agent, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockAgentRepo) GetByUserAndMachine(ctx context.Context, userID, machineID string) (*models.Agent, error) {
	return m.getByUserAndMachineFn(ctx, userID, machineID)
}

func (m *mockAgentRepo) GetByToken(ctx context.Context, token string) (*models.Agent, error) {
	return m.getByTokenFn(ctx, token)
}

func (m *mockAgentRepo) Create(ctx context.Context, userID, token, name, machineID string) (*models.Agent, error) {
	return m.createFn(ctx, userID, token, name, machineID)
}

func (m *mockAgentRepo) ListByUser(ctx context.Context, userID string) ([]models.Agent, error) {
	return m.listByUserFn(ctx, userID)
}

func (m *mockAgentRepo) UpdateLastSeen(ctx context.Context, agentID string) error {
	return m.updateLastSeenFn(ctx, agentID)
}

// TESTS
func TestRegisterOrGet_NoExistingAgent(t *testing.T) {
	repo := &mockAgentRepo{
		getByUserAndMachineFn: func(ctx context.Context, userID, machineID string) (*models.Agent, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, userID, token, name, machineID string) (*models.Agent, error) {
			return &models.Agent{
				UserID:    userID,
				Token:     token,
				Name:      name,
				MachineID: machineID,
			}, nil
		},
	}

	svc := &AgentService{
		agentRepo: repo,
	}

	agent, err := svc.RegisterOrGet(context.Background(), "1", "test", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agent == nil {
		t.Fatalf("No agent registered")
	}

	if agent.Token == "" {
		t.Errorf("No token generated for agent")
	}

	if agent.Name != "test" {
		t.Errorf("Incorrect agent name")
	}
	if agent.UserID != "1" {
		t.Errorf("Incorrect agent userID")
	}

	if agent.MachineID != "1" {
		t.Errorf("Incorrect agent machineID")
	}
}

func TestRegisterOrGet_ExistingAgent(t *testing.T) {
	createCalled := false
	repo := &mockAgentRepo{
		getByUserAndMachineFn: func(ctx context.Context, userID, machineID string) (*models.Agent, error) {
			return &models.Agent{
				UserID:    userID,
				Token:     "token1",
				Name:      "test",
				MachineID: machineID,
			}, nil
		},

		createFn: func(ctx context.Context, userID, token, name, machineID string) (*models.Agent, error) {
			createCalled = true
			return nil, nil
		},
	}

	svc := &AgentService{
		agentRepo: repo,
	}

	agent, err := svc.RegisterOrGet(context.Background(), "1", "test", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agent == nil {
		t.Fatalf("No agent existed")
	}

	if agent.Token == "" {
		t.Errorf("No token for agent")
	}

	if agent.Name != "test" {
		t.Errorf("Incorrect agent name")
	}
	if agent.UserID != "1" {
		t.Errorf("Incorrect agent userID")
	}

	if agent.MachineID != "1" {
		t.Errorf("Incorrect agent machineID")
	}

	if createCalled {
		t.Errorf("Create method called when exists")
	}
}

func TestRegisterOrGet_DBError_OnLookup(t *testing.T) {
	repo := &mockAgentRepo{
		getByUserAndMachineFn: func(ctx context.Context, userID, machineID string) (*models.Agent, error) {
			return nil, fmt.Errorf("DB error while getting agent")
		},
	}

	svc := &AgentService{
		agentRepo: repo,
	}

	agent, err := svc.RegisterOrGet(context.Background(), "1", "test", "1")
	if agent != nil {
		t.Errorf("Agent is not nil")
	}

	if err == nil || !strings.Contains(err.Error(), "DB error") {
		t.Errorf("expected DB error, got: %v", err)
	}
}

func TestRegisterOrGet_DBError_OnCreate(t *testing.T) {
	repo := &mockAgentRepo{
		getByUserAndMachineFn: func(ctx context.Context, userID, machineID string) (*models.Agent, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, userID, token, name, machineID string) (*models.Agent, error) {
			return nil, fmt.Errorf("DB error while creating agent")
		},
	}

	svc := &AgentService{
		agentRepo: repo,
	}

	agent, err := svc.RegisterOrGet(context.Background(), "1", "test", "1")
	if agent != nil {
		t.Errorf("Agent is not nil")
	}

	if err == nil || !strings.Contains(err.Error(), "DB error") {
		t.Errorf("expected DB error, got: %v", err)
	}
}

func TestGetByID_Found(t *testing.T) {
	repo := &mockAgentRepo{
		getByIDFn: func(ctx context.Context, id string) (*models.Agent, error) {
			return &models.Agent{
				Name:      "test",
				Token:     "token1",
				ID:        "1",
				UserID:    id,
				MachineID: "1",
			}, nil
		},
	}

	svc := &AgentService{agentRepo: repo}

	agent, err := svc.GetByID(context.Background(), "1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatalf("Agent is nil")
	}

	if agent.Name != "test" {
		t.Errorf("Incorrect agent name")
	}

	if agent.UserID != "1" {
		t.Errorf("Incorrect agent UserID")
	}

	if agent.MachineID != "1" {
		t.Errorf("Incorrect agent MachineID")
	}
}

func TestListByUser_ReturnsList(t *testing.T) {
	expectedAgents := []models.Agent{
		{Name: "test1", Token: "token1", MachineID: "1", UserID: "1"},
		{Name: "test2", Token: "token2", MachineID: "2", UserID: "1"},
	}
	repo := &mockAgentRepo{
		listByUserFn: func(ctx context.Context, userID string) ([]models.Agent, error) {

			return expectedAgents, nil
		},
	}

	svc := &AgentService{agentRepo: repo}

	agents, err := svc.ListByUser(context.Background(), "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(agents) != len(expectedAgents) {
		t.Errorf("Invalid array len, expected %d got: %d", len(expectedAgents), len(agents))
	}

	for i, agent := range agents {
		if agent.Name != expectedAgents[i].Name {
			t.Errorf("Incorrect agent name, expected %s got: %s", expectedAgents[i].Name, agent.Name)
		}

		if agent.Token != expectedAgents[i].Token {
			t.Errorf("Incorrect agent token, expected %s got: %s", expectedAgents[i].Token, agent.Token)
		}

		if agent.MachineID != expectedAgents[i].MachineID {
			t.Errorf("Incorrect agent machineID, expected %s got: %s", expectedAgents[i].MachineID, agent.MachineID)
		}

		if agent.UserID != "1" {
			t.Errorf("Incorrect userID")
		}
	}
}
