package http

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/dragodui/my-deploy/internal/http/middleware"
	"github.com/dragodui/my-deploy/internal/registry"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSHandler struct {
	registry *registry.AgentRegistry
}

func NewWSHandler(reg *registry.AgentRegistry) *WSHandler {
	return &WSHandler{registry: reg}
}

func (h *WSHandler) HandleAgentWS(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	ac := h.registry.Register(userID, conn)
	log.Printf("agent connected: userID=%s", userID)

	defer func() {
		h.registry.Unregister(userID)
		conn.Close()
		log.Printf("agent disconnected: userID=%s", userID)
	}()

	// read loop: receive results from agent
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

		// check message type
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
		}
	}
}
