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
		return
	}

	fmt.Println("Authenticated. Ready to go!")
}
