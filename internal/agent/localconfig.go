package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const DefaultServerURL = "http://localhost:8080"

type LocalConfig struct {
	UserName string `json:"user_name"`
	AgentName string `json:"agent_name"`
	URL        string `json:"url"`
	AgentToken string `json:"agent_token"`
	JWT        string `json:"jwt"`
	MachineID  string `json:"machine_id"`
	DockerHost string `json:"docker_host"`
}

func Load() (*LocalConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(home, ".mydeploy", "config.json")
	configStr, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	var config LocalConfig

	if err := json.Unmarshal(configStr, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func Save(config *LocalConfig) error {
	config.URL = DefaultServerURL

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(home, ".mydeploy"), 0700); err != nil {
		return err
	}
	configPath := filepath.Join(home, ".mydeploy", "config.json")

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, configJSON, 0600); err != nil {
		return err
	}

	return nil
}
