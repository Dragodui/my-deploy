package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/dragodui/my-deploy/internal/http/middleware"
	"github.com/dragodui/my-deploy/internal/models"
)

type AgentServicer interface {
	RegisterOrGet(ctx context.Context, userID, name, machineID string) (*models.Agent, error)
	ListByUser(ctx context.Context, userID string) ([]models.Agent, error)
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

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
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

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	agents, err := h.svc.ListByUser(r.Context(), userID)
	if err != nil {
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
