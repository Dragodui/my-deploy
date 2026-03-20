package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"

	"github.com/dragodui/my-deploy/internal/gateway"
	"github.com/dragodui/my-deploy/internal/shared/auth"
)

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

// jwtToUserID validates JWT and sets X-User-ID header for downstream services
func jwtToUserID(jwtSecret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		userID, err := auth.ValidateToken(parts[1], jwtSecret)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		r.Header.Set("X-User-ID", userID)
		next.ServeHTTP(w, r)
	})
}

func main() {
	cfg := gateway.LoadConfig()

	authProxy := httputil.NewSingleHostReverseProxy(cfg.AuthURL)
	agentProxy := httputil.NewSingleHostReverseProxy(cfg.AgentURL)
	deployProxy := httputil.NewSingleHostReverseProxy(cfg.DeployURL)

	mux := http.NewServeMux()

	// health
	mux.HandleFunc("GET /health", healthCheck)

	// auth routes
	mux.Handle("/api/auth/", authProxy)
	mux.Handle("GET /api/me", jwtToUserID(cfg.JWTSecret, authProxy))

	// agent routes
	mux.Handle("POST /api/agent", jwtToUserID(cfg.JWTSecret, agentProxy))
	mux.Handle("GET /api/agents", jwtToUserID(cfg.JWTSecret, agentProxy))

	// agent websocket
	mux.Handle("GET /ws/agent", gateway.WSProxy(cfg.AgentURL))

	// deploy routes
	mux.Handle("POST /api/deployments", jwtToUserID(cfg.JWTSecret, deployProxy))
	mux.Handle("GET /api/deployments", jwtToUserID(cfg.JWTSecret, deployProxy))
	mux.Handle("GET /api/deployments/", jwtToUserID(cfg.JWTSecret, deployProxy))
	mux.Handle("DELETE /api/deployments/", jwtToUserID(cfg.JWTSecret, deployProxy))
	mux.Handle("POST /api/deployments/", jwtToUserID(cfg.JWTSecret, deployProxy))

	// deploy logs websocket
	mux.Handle("GET /ws/logs/", gateway.WSProxy(cfg.DeployURL))

	log.Printf("gateway starting on port %d", cfg.Port)
	http.ListenAndServe(":"+strconv.Itoa(cfg.Port), mux)
}
