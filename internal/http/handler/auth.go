package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/dragodui/my-deploy/internal/http/middleware"
	"github.com/dragodui/my-deploy/internal/models"
)

type AuthServicer interface {
	SignUp(ctx context.Context, email, name, password string) (string, string, error)
	SignIn(ctx context.Context, email, password string) (string, string, error)
	Me(ctx context.Context, userID string) (*models.User, error)
}

type AuthHandler struct {
	svc AuthServicer
}

func NewAuthHandler(svc AuthServicer) *AuthHandler {
	return &AuthHandler{
		svc: svc,
	}
}

func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	type signUpReq struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	var req signUpReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, name, err := h.svc.SignUp(
		r.Context(),
		req.Email,
		req.Name,
		req.Password,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	type response struct {
		Token string `json:"token"`
		Name  string `json:"name"`
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response{
		Token: token,
		Name:  name,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		log.Printf("[ERROR] auth.Me: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	type response struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	})
}

func (h *AuthHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	type signInReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req signInReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, name, err := h.svc.SignIn(
		r.Context(),
		req.Email,
		req.Password,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	type response struct {
		Token string `json:"token"`
		Name  string `json:"name"`
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response{
		Token: token,
		Name:  name,
	})
}
