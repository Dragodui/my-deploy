package agentsvc

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSHandler struct {
	registry *AgentRegistry
	repo     *AgentRepository
}

func NewWSHandler(reg *AgentRegistry, repo *AgentRepository) *WSHandler {
	return &WSHandler{registry: reg, repo: repo}
}

func (h *WSHandler) HandleAgentWS(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Agent-Token")
	if token == "" {
		http.Error(w, "missing agent token", http.StatusUnauthorized)
		return
	}

	ag, err := h.repo.GetByToken(r.Context(), token)
	if err != nil || ag == nil {
		http.Error(w, "invalid agent token", http.StatusUnauthorized)
		return
	}

	agentID := ag.ID

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	ac := h.registry.Register(agentID, conn)
	log.Printf("agent connected: %s", agentID)

	defer func() {
		h.registry.Unregister(agentID)
		conn.Close()
		log.Printf("agent disconnected: %s", agentID)
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("agent read error: %v", err)
			}
			return
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(msg, &raw); err != nil {
			continue
		}

		var msgType string
		if t, ok := raw["type"]; ok {
			json.Unmarshal(t, &msgType)
		}

		switch msgType {
		case "ping":
			pong, _ := json.Marshal(map[string]string{"type": "pong"})
			conn.WriteMessage(websocket.TextMessage, pong)
		case "result":
			var result agent.Result
			if err := json.Unmarshal(msg, &result); err != nil {
				log.Printf("invalid result: %v", err)
				continue
			}
			ac.HandleResult(result)
		case "progress":
			var prog agent.Progress
			if err := json.Unmarshal(msg, &prog); err != nil {
				log.Printf("invalid progress: %v", err)
				continue
			}
			ac.HandleProgress(prog)
		case "logs":
			var chunk agent.LogChunk
			if err := json.Unmarshal(msg, &chunk); err != nil {
				log.Printf("invalid log chunk: %v", err)
				continue
			}
			ac.HandleLogChunk(chunk)
		}
	}
}
