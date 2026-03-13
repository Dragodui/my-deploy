package main

import (
	"context"
	"flag"
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
	token := flag.String("token", "", "agent token")
	url := flag.String("url", "", "server URL")
	flag.Parse()
	var config *agent.LocalConfig
	var err error

	if *token != "" && *url != "" {
		config = &agent.LocalConfig{
			AgentToken: *token,
			URL:        *url,
		}
		agent.Save(config)
	} else {
		config, err = agent.Load()
		if err != nil || config == nil {
			fmt.Println("No config found. Run with --token and --url")
			os.Exit(1)
		}
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

	log.Printf("agent starting, connecting to %s", config.URL)
	a.Run(ctx)
}
