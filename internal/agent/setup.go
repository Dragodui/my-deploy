package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func prompt(scanner *bufio.Scanner, label string) string {
	fmt.Print(label + " ")
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func Setup(api *APIClient) (*LocalConfig, error) {
	fmt.Println("----- MyDeploy Setup -----")
	scanner := bufio.NewScanner(os.Stdin)

	// email
	var email string
	for email == "" {
		email = prompt(scanner, "Email:")
		if email == "" {
			fmt.Println("Email is required")
		}
	}

	// password
	var password string
	for len(password) < 8 {
		password = prompt(scanner, "Password:")
		if len(password) < 8 {
			fmt.Println("Password must be at least 8 characters")
		}
	}

	// sign in or sign up
	var jwt string
	for {
		choice := prompt(scanner, "Sign in or Sign up? [in/up]:")
		switch choice {
		case "in":
			token, err := api.SignIn(email, password)
			if err != nil {
				fmt.Printf("Sign in failed: %v. Try again.\n", err)
				continue
			}
			jwt = token
		case "up":
			name := prompt(scanner, "Your name:")
			token, err := api.SignUp(email, name, password)
			if err != nil {
				fmt.Printf("Sign up failed: %v. Try again.\n", err)
				continue
			}
			jwt = token
		default:
			fmt.Println("Enter 'in' or 'up'")
			continue
		}
		break
	}
	fmt.Println("Authenticated successfully")

	// agent name
	hostname, _ := os.Hostname()
	agentName := prompt(scanner, fmt.Sprintf("Agent name (default: %s):", hostname))
	if agentName == "" {
		agentName = hostname
	}

	// docker host
	dockerHost := prompt(scanner, "Docker host (enter for default):")

	// register agent
	machineID := GenerateMachineID()
	agentToken, err := api.RegisterAgent(jwt, agentName, machineID)
	if err != nil {
		return nil, fmt.Errorf("agent registration failed: %w", err)
	}
	fmt.Println("Agent registered successfully")

	config := &LocalConfig{
		URL:        DefaultServerURL,
		JWT:        jwt,
		MachineID:  machineID,
		AgentToken: agentToken,
		DockerHost: dockerHost,
	}

	if err := Save(config); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Println("Config saved")

	return config, nil
}
