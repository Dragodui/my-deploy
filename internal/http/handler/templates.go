package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/dragodui/my-deploy/internal/models"
)

type TemplatesServicer interface {
	GetAll(ctx context.Context) []*models.AppTemplate
}

type TemplatesHandler struct {
	svc TemplatesServicer
}

func NewTemplatesHandler(svc TemplatesServicer) *TemplatesHandler {
	return &TemplatesHandler{svc}
}

func (h *TemplatesHandler) GetAll(w http.ResponseWriter, r *http.Request) {

	templates := h.svc.GetAll(r.Context())

	type response struct {
		Templates []*models.AppTemplate `json:"templates"`
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response{
		Templates: templates,
	})
}
