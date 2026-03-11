package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/moby/moby/client"
)

func main() {
	apiClient := agent.NewAPIClient(agent.DefaultServerURL)
	config, err := agent.Load()
	if err != nil {
		fmt.Printf("Error in config setup: %v", err)
		return
	}
	if config == nil {
		config, err = agent.Setup(apiClient)
	}

	if err != nil {
		fmt.Printf("Error in config setup: %v", err)
		return
	}

	opts := []client.Opt{}
	if config.DockerHost != "" {
		opts = append(opts, client.WithHost(config.DockerHost))
	}
	dockerClient, err := client.New(opts...)
	if err != nil {
		log.Fatalf("failed to create docker client: %v", err)
	}

	handler := agent.NewHandler(dockerClient)
	wsURL := strings.Replace(config.URL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = wsURL + "/ws/agent"
	a := agent.New(wsURL, config.AgentToken, handler)

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

	log.Printf("agent starting, connecting to %s", agent.DefaultServerURL)
	a.Run(ctx)
}
