package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/gorilla/websocket"
)

type AgentConn struct {
	conn    *websocket.Conn
	mu      sync.Mutex
	pending map[string]chan agent.Result
}

func newAgentConn(conn *websocket.Conn) *AgentConn {
	return &AgentConn{
		conn:    conn,
		pending: make(map[string]chan agent.Result),
	}
}

// SendCommand sends a command to the agent and waits for a result.
func (ac *AgentConn) SendCommand(ctx context.Context, cmd agent.Command) (agent.Result, error) {
	ch := make(chan agent.Result, 1)

	ac.mu.Lock()
	ac.pending[cmd.ID] = ch
	ac.mu.Unlock()

	defer func() {
		ac.mu.Lock()
		delete(ac.pending, cmd.ID)
		ac.mu.Unlock()
	}()

	data, err := json.Marshal(cmd)
	if err != nil {
		return agent.Result{}, err
	}

	ac.mu.Lock()
	err = ac.conn.WriteMessage(websocket.TextMessage, data)
	ac.mu.Unlock()
	if err != nil {
		return agent.Result{}, fmt.Errorf("failed to send command: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	select {
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		return agent.Result{}, fmt.Errorf("timeout waiting for agent response")
	}
}

// HandleResult routes a result to the waiting sender.
func (ac *AgentConn) HandleResult(result agent.Result) {
	ac.mu.Lock()
	ch, ok := ac.pending[result.ID]
	ac.mu.Unlock()

	if ok {
		ch <- result
	}
}

func (ac *AgentConn) Close() {
	ac.conn.Close()
}

// AgentRegistry tracks connected agents by token.
type AgentRegistry struct {
	mu     sync.RWMutex
	agents map[string]*AgentConn // key = agent token
}

func New() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*AgentConn),
	}
}

func (r *AgentRegistry) Register(token string, conn *websocket.Conn) *AgentConn {
	ac := newAgentConn(conn)

	r.mu.Lock()
	// close old connection if exists
	if old, ok := r.agents[token]; ok {
		old.Close()
	}
	r.agents[token] = ac
	r.mu.Unlock()

	return ac
}

func (r *AgentRegistry) Unregister(token string) {
	r.mu.Lock()
	delete(r.agents, token)
	r.mu.Unlock()
}

func (r *AgentRegistry) Get(token string) (*AgentConn, bool) {
	r.mu.RLock()
	ac, ok := r.agents[token]
	r.mu.RUnlock()
	return ac, ok
}
