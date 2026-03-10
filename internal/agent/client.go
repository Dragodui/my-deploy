package agent

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Agent struct {
	serverURL string
	token     string
	handler   *Handler
}

func New(serverURL, token string, handler *Handler) *Agent {
	return &Agent{
		serverURL: serverURL,
		token:     token,
		handler:   handler,
	}
}

// Run connects to the server and reconnects on disconnect.
func (a *Agent) Run(ctx context.Context) {
	backoff := time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := a.connect(ctx)
		if err != nil {
			log.Printf("disconnected: %v, reconnecting in %s...", err, backoff)
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
	}
}

func (a *Agent) connect(ctx context.Context) error {
	header := make(map[string][]string)
	header["Authorization"] = []string{"Bearer " + a.token}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, a.serverURL, header)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Println("connected to server")

	// ping loop
	go a.pingLoop(ctx, conn)

	// read loop
	for {
		select {
		case <-ctx.Done():
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return ctx.Err()
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var cmd Command
		if err := json.Unmarshal(msg, &cmd); err != nil {
			log.Printf("invalid command: %v", err)
			continue
		}

		if cmd.Type == "pong" {
			continue
		}

		// handle command in goroutine so we don't block reads
		go func(cmd Command) {
			result := a.handler.Handle(ctx, cmd)

			data, _ := json.Marshal(result)
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("failed to send result: %v", err)
			}
		}(cmd)
	}
}

func (a *Agent) pingLoop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ping := map[string]string{"type": "ping"}
			data, _ := json.Marshal(ping)
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}
}
