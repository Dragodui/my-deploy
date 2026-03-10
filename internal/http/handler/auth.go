package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dragodui/my-deploy/internal/service"
)

type AuthHandler struct {
	svc service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{
		svc: *svc,
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

	token, err := h.svc.SignUp(
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
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response{
		Token: token,
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

	token, err := h.svc.SignIn(
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
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response{
		Token: token,
	})
}