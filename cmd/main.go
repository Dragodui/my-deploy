package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/dragodui/my-deploy/cmd/server"
	"github.com/dragodui/my-deploy/internal/config"
)

func main() {
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatalf("failed to create logs directory: %v", err)
	}

	logFile, err := os.OpenFile("logs/server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg := config.NewConfig()
	srv := server.NewServer(cfg)
	log.Printf("server starting on port %d", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
