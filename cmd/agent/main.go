package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/moby/moby/client"
)

func main() {
	serverURL := flag.String("server", "ws://localhost:8080/ws/agent", "server websocket url")
	token := flag.String("token", "", "auth token")
	dockerHost := flag.String("docker-host", "", "docker host (default: local socket)")
	flag.Parse()

	if *token == "" {
		log.Fatal("--token is required")
	}

	// connect to local docker
	opts := []client.Opt{}
	if *dockerHost != "" {
		opts = append(opts, client.WithHost(*dockerHost))
	}

	dockerClient, err := client.New(opts...)
	if err != nil {
		log.Fatalf("failed to create docker client: %v", err)
	}

	handler := agent.NewHandler(dockerClient)
	a := agent.New(*serverURL, *token, handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("shutting down agent...")
		cancel()
	}()

	log.Printf("agent starting, connecting to %s", *serverURL)
	a.Run(ctx)
}
