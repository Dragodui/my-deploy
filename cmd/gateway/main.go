package main

import (
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/dragodui/my-deploy/internal/gateway"
	"github.com/dragodui/my-deploy/internal/shared/http/middleware"
)

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func main() {
	cfg := gateway.LoadConfig()

	authProxy := httputil.NewSingleHostReverseProxy(cfg.AuthURL)
	mux := http.NewServeMux()

	// без JWT
	mux.Handle("/api/auth/", authProxy)
	mux.HandleFunc("/health", healthCheck)
	mux.Handle("/api/me", middleware.JWTAuth(cfg.JWTSecret)(authProxy))

	http.ListenAndServe(":"+strconv.Itoa(cfg.Port), mux)
}
