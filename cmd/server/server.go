package server

import (
	"net/http"
)

func NewServer() *http.ServeMux {

	server := http.NewServeMux()
	server.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	return server
}
