package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dragodui/my-deploy/cmd/server"
	"github.com/dragodui/my-deploy/internal/config"
)

func main() {
	cfg := config.NewConfig()
	srv := server.NewServer(cfg)
	log.Printf("server starting on port %d", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
