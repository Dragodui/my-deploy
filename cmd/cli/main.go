package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/dragodui/my-deploy/internal/cli"
)

func main() {
	config, err := agent.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	api := agent.NewAPIClient(agent.DefaultServerURL)

	// if no config or jwt expired => auth
	needsAuth := config == nil || agent.IsJWTExpired(config.JWT)

	if needsAuth {
		model := cli.NewAuthModel(api)
		if _, err := tea.NewProgram(model).Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		// reload config after auth (now has JWT)
		config, err = agent.Load()
		if err != nil || config == nil {
			log.Fatalf("Error loading config after auth: %v", err)
		}
	}

	if config.AgentToken == "" {
		agentModel := cli.NewAgentCreateModel(api)
		if _, err := tea.NewProgram(agentModel).Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("Ready to go!")
}
