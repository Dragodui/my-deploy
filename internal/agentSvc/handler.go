package agentsvc

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/dragodui/my-deploy/internal/shared/models"
)

type AgentServicer interface {
	RegisterOrGet(ctx context.Context, userID, name, machineID string) (*models.Agent, error)
	ListByUser(ctx context.Context, userID string) ([]models.Agent, error)
	CreateBootstrapToken(ctx context.Context, userID, agentName string) (*models.AgentBootstrapToken, error)
	ExchangeBootstrapToken(ctx context.Context, token, machineID string) (*models.Agent, error)
}

type AgentHandler struct {
	svc AgentServicer
}

func NewAgentHandler(svc AgentServicer) *AgentHandler {
	return &AgentHandler{
		svc: svc,
	}
}

func (h *AgentHandler) RegisterOrGet(w http.ResponseWriter, r *http.Request) {

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	type getAgentReq struct {
		Name      string `json:"name"`
		MachineID string `json:"machine_id"`
	}
	var req getAgentReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.MachineID == "" {
		http.Error(w, "name and machine_id are required", http.StatusBadRequest)
		return
	}

	agent, err := h.svc.RegisterOrGet(r.Context(), userID, req.Name, req.MachineID)
	if err != nil {
		log.Printf("[ERROR] agent.RegisterOrGet: %v", err)
		http.Error(w, "failed to register agent", http.StatusInternalServerError)
		return
	}

	type response struct {
		Agent *models.Agent `json:"agent"`
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response{
		Agent: agent,
	})
}

func (h *AgentHandler) ListByUser(w http.ResponseWriter, r *http.Request) {

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	agents, err := h.svc.ListByUser(r.Context(), userID)
	if err != nil {
		log.Printf("[ERROR] agent.ListByUser: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	type response struct {
		Agents []models.Agent `json:"agents"`
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response{
		Agents: agents,
	})

}

func (h *AgentHandler) CreateBootstrapToken(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	type request struct {
		Name string `json:"name"`
	}
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	bt, err := h.svc.CreateBootstrapToken(r.Context(), userID, req.Name)
	if err != nil {
		log.Printf("[ERROR] agent.CreateBootstrapToken: %v", err)
		http.Error(w, "failed to create bootstrap token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Token     string `json:"token"`
		AgentName string `json:"agent_name"`
		ExpiresAt string `json:"expires_at"`
	}{
		Token:     bt.Token,
		AgentName: bt.AgentName,
		ExpiresAt: bt.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *AgentHandler) ExchangeBootstrapToken(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Token     string `json:"token"`
		MachineID string `json:"machine_id"`
	}
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Token == "" || req.MachineID == "" {
		http.Error(w, "token and machine_id are required", http.StatusBadRequest)
		return
	}

	ag, err := h.svc.ExchangeBootstrapToken(r.Context(), req.Token, req.MachineID)
	if err != nil {
		log.Printf("[ERROR] agent.ExchangeBootstrapToken: %v", err)
		http.Error(w, "failed to exchange bootstrap token", http.StatusInternalServerError)
		return
	}
	if ag == nil {
		http.Error(w, "invalid or expired bootstrap token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Agent *models.Agent `json:"agent"`
	}{
		Agent: ag,
	})
}
