package agentsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dragodui/my-deploy/internal/shared/models"
)

type mockAgentService struct {
	registerOrGetFn func(ctx context.Context, userID, name, machineID string) (*models.Agent, error)
	listByUserFn    func(ctx context.Context, userID string) ([]models.Agent, error)
}

func (m *mockAgentService) RegisterOrGet(ctx context.Context, userID, name, machineID string) (*models.Agent, error) {
	return m.registerOrGetFn(ctx, userID, name, machineID)
}

func (m *mockAgentService) ListByUser(ctx context.Context, userID string) ([]models.Agent, error) {
	return m.listByUserFn(ctx, userID)
}

func TestHandler_RegisterOrGet(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		body       string
		mockFn     func(ctx context.Context, userID, name, machineID string) (*models.Agent, error)
		wantStatus int
	}{
		{
			name:   "success",
			userID: "user1",
			body:   `{"name":"test","machine_id":"m1"}`,
			mockFn: func(ctx context.Context, userID, name, machineID string) (*models.Agent, error) {
				return &models.Agent{ID: "a1", Name: name}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing X-User-ID",
			userID:     "",
			body:       `{"name":"test","machine_id":"m1"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid json",
			userID:     "user1",
			body:       `not json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing name",
			userID:     "user1",
			body:       `{"machine_id":"m1"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing machine_id",
			userID:     "user1",
			body:       `{"name":"test"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: "user1",
			body:   `{"name":"test","machine_id":"m1"}`,
			mockFn: func(ctx context.Context, userID, name, machineID string) (*models.Agent, error) {
				return nil, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAgentService{registerOrGetFn: tt.mockFn}
			h := NewAgentHandler(svc)

			req := httptest.NewRequest("POST", "/api/agent", bytes.NewBufferString(tt.body))
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}

			rec := httptest.NewRecorder()

			h.RegisterOrGet(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d; want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestHandler_ListByUser(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		mockFn     func(ctx context.Context, userID string) ([]models.Agent, error)
		wantStatus int
		wantAgents int
	}{
		{
			name:   "success",
			userID: "1",
			mockFn: func(ctx context.Context, userID string) ([]models.Agent, error) {
				return []models.Agent{
					models.Agent{
						Name:      "test1",
						Token:     "token1",
						MachineID: "1",
						UserID:    userID,
					},
					models.Agent{
						Name:      "test2",
						Token:     "token2",
						MachineID: "2",
						UserID:    userID,
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantAgents: 2,
		},
		{
			name:   "empty list",
			userID: "1",
			mockFn: func(ctx context.Context, userID string) ([]models.Agent, error) {
				return []models.Agent{}, nil
			},
			wantStatus: http.StatusOK,
			wantAgents: 0,
		},
		{
			name:       "missing X-User-ID",
			userID:     "",
			wantStatus: http.StatusUnauthorized,
			wantAgents: -1,
		},
		{name: "service error", userID: "1", mockFn: func(ctx context.Context, userID string) ([]models.Agent, error) {
			return nil, fmt.Errorf("Error while getting agents")
		}, wantStatus: http.StatusInternalServerError, wantAgents: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAgentService{listByUserFn: tt.mockFn}
			h := NewAgentHandler(svc)

			req := httptest.NewRequest("GET", "/api/agents", nil)
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}

			rec := httptest.NewRecorder()

			h.ListByUser(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d; want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if tt.wantAgents >= 0 {
				var resp struct {
					Agents []models.Agent `json:"agents"`
				}

				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if len(resp.Agents) != tt.wantAgents {
					t.Errorf("got %d agents; want %d", len(resp.Agents), tt.wantAgents)
				}
			}
		})
	}
}
