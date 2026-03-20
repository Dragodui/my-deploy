package deploysvc

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/dragodui/my-deploy/internal/shared/models"
	agentpb "github.com/dragodui/my-deploy/internal/shared/proto/agentpb/proto"
)

type DeployServicer interface {
	Create(ctx context.Context, agentID string, req models.DeployRequest) (*models.Deployment, error)
	GetByID(ctx context.Context, id string) (*models.Deployment, error)
	ListByAgent(ctx context.Context, agentID string) ([]models.Deployment, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateContainerID(ctx context.Context, id, containerID string) error
	Delete(ctx context.Context, id string) error
	GetProgress(ctx context.Context, deployID string) string
	InspectDeployment(ctx context.Context, id string) (string, error)
	Stop(ctx context.Context, containerID, agentID string) error
	Start(ctx context.Context, containerID, agentID string) error
}

type DeployHandler struct {
	svc         DeployServicer
	agentClient agentpb.AgentInternalClient
}

func NewDeployHandler(svc DeployServicer, agentClient agentpb.AgentInternalClient) *DeployHandler {
	return &DeployHandler{svc: svc, agentClient: agentClient}
}

func (h *DeployHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	type createReq struct {
		AgentID string `json:"agent_id"`
		models.DeployRequest
	}
	var req createReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.AgentID == "" || req.Name == "" {
		http.Error(w, "agent_id and name are required", http.StatusBadRequest)
		return
	}

	var agentReq agentpb.GetAgentRequest = agentpb.GetAgentRequest{
		Id: req.AgentID,
	}
	ag, err := h.agentClient.GetAgent(r.Context(), &agentReq)
	if err != nil {
		log.Printf("[ERROR] deploy.Create agent lookup: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if ag == nil || ag.UserId != userID {
		http.Error(w, "agent not found", http.StatusForbidden)
		return
	}

	deploy, err := h.svc.Create(r.Context(), req.AgentID, req.DeployRequest)
	if err != nil {
		http.Error(w, "failed to create deployment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	type response struct {
		Deployment models.Deployment `json:"deployment"`
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response{
		Deployment: *deploy,
	})
}

func (h *DeployHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	deploy, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("[ERROR] deploy.GetByID: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if deploy == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if deploy.Status == "deploying" {
		deploy.Progress = h.svc.GetProgress(r.Context(), id)
	} else if deploy.ContainerID != nil {
		if status, err := h.svc.InspectDeployment(r.Context(), id); err == nil {
			deploy.Status = status
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deploy)
}

func (h *DeployHandler) ListByAgent(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "agent_id query param is required", http.StatusBadRequest)
		return
	}

	deployments, err := h.svc.ListByAgent(r.Context(), agentID)
	if err != nil {
		log.Printf("[ERROR] deploy.ListByAgent: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deployments)
}

func (h *DeployHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		http.Error(w, "failed to delete deployment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DeployHandler) Stop(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	deploy, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if deploy.AgentID == "" {
		http.Error(w, "no agent for this deploy found", http.StatusBadRequest)
		return
	}

	if deploy.ContainerID == nil || *deploy.ContainerID == "" {
		http.Error(w, "no container for this deploy found", http.StatusBadRequest)
		return
	}
	agentReq := agentpb.GetAgentRequest{
		Id: deploy.AgentID,
	}
	ag, err := h.agentClient.GetAgent(r.Context(), &agentReq)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if ag == nil || ag.UserId != userID {
		http.Error(w, "agent not found", http.StatusForbidden)
		return
	}

	if err := h.svc.Stop(r.Context(), *deploy.ContainerID, deploy.AgentID); err != nil {
		http.Error(w, "failed to stop deployment", http.StatusInternalServerError)
		return
	}

	if err := h.svc.UpdateStatus(r.Context(), id, "exited"); err != nil {
		http.Error(w, "failed to update deployment status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
func (h *DeployHandler) Start(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	deploy, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if deploy.AgentID == "" {
		http.Error(w, "no agent for this deploy found", http.StatusBadRequest)
		return
	}

	if deploy.ContainerID == nil || *deploy.ContainerID == "" {
		http.Error(w, "no container for this deploy found", http.StatusBadRequest)
		return
	}
	agentReq := agentpb.GetAgentRequest{
		Id: deploy.AgentID,
	}
	ag, err := h.agentClient.GetAgent(r.Context(), &agentReq)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if ag == nil || ag.UserId != userID {
		http.Error(w, "agent not found", http.StatusForbidden)
		return
	}

	if err := h.svc.Start(r.Context(), *deploy.ContainerID, deploy.AgentID); err != nil {
		http.Error(w, "failed to start deployment", http.StatusInternalServerError)
		return
	}

	if err := h.svc.UpdateStatus(r.Context(), id, "running"); err != nil {
		http.Error(w, "failed to update deployment status", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
