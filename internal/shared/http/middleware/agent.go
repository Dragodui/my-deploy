package middleware

import (
	"context"
	"net/http"

	"github.com/dragodui/my-deploy/internal/shared/models"
)

const AgentIDKey contextKey = "agent_id"

type AgentRepo interface {
	GetByToken(ctx context.Context, token string) (*models.Agent, error)
}

func AgentAuth(repo AgentRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Agent-Token")
			if token == "" {
				http.Error(w, "missing agent token", http.StatusUnauthorized)
				return
			}

			agent, err := repo.GetByToken(r.Context(), token)
			if agent == nil && err == nil {
				http.Error(w, "invalid agent token", http.StatusUnauthorized)
				return
			}
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), AgentIDKey, agent.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AgentIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(AgentIDKey).(string)
	return id, ok
}
