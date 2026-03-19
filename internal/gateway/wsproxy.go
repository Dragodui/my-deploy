package gateway

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSProxy proxies a WebSocket connection to a backend service.
// It upgrades the client connection, dials the backend, and pipes messages both ways.
func WSProxy(backendURL *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// build backend ws url
		scheme := "ws"
		if backendURL.Scheme == "https" {
			scheme = "wss"
		}
		backendWS := scheme + "://" + backendURL.Host + r.URL.Path

		// forward headers from client to backend
		reqHeader := http.Header{}
		for _, h := range []string{"X-Agent-Token", "Authorization", "X-User-ID"} {
			if v := r.Header.Get(h); v != "" {
				reqHeader.Set(h, v)
			}
		}

		// forward subprotocols
		if protos := r.Header.Get("Sec-WebSocket-Protocol"); protos != "" {
			for _, p := range strings.Split(protos, ",") {
				reqHeader.Add("Sec-WebSocket-Protocol", strings.TrimSpace(p))
			}
		}

		// dial backend
		backendConn, _, err := websocket.DefaultDialer.Dial(backendWS, reqHeader)
		if err != nil {
			log.Printf("ws proxy: failed to dial backend %s: %v", backendWS, err)
			http.Error(w, "backend unavailable", http.StatusBadGateway)
			return
		}
		defer backendConn.Close()

		// upgrade client
		clientConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws proxy: upgrade failed: %v", err)
			return
		}
		defer clientConn.Close()

		// pipe: client → backend
		go func() {
			for {
				msgType, data, err := clientConn.ReadMessage()
				if err != nil {
					backendConn.WriteMessage(websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
					return
				}
				if err := backendConn.WriteMessage(msgType, data); err != nil {
					return
				}
			}
		}()

		// pipe: backend → client
		for {
			msgType, data, err := backendConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("ws proxy: backend closed: %v", err)
				}
				clientConn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := clientConn.WriteMessage(msgType, data); err != nil {
				return
			}
		}
	})
}

