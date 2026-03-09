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
	server := server.NewServer(cfg)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), server); err != nil {
		log.Fatalf("Error while launching the server: %v", err)
	}

	log.Printf("Server runs on port %d\n", cfg.Port)
}
