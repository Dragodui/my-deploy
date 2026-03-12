package agent

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dragodui/my-deploy/internal/models"
)

func IsJWTExpired(token string) bool {
	if token == "" {
		return true
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return true
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return true
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}

	if err := json.Unmarshal(payload, &claims); err != nil {
		return true
	}

	return time.Now().Unix() > claims.Exp
}

type APIClient struct {
	ServerURL  string
	HTTPClient *http.Client
}

func NewAPIClient(url string) *APIClient {
	return &APIClient{
		ServerURL:  url,
		HTTPClient: &http.Client{},
	}
}

func (api *APIClient) SignIn(email, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})

	resp, err := api.HTTPClient.Post(api.ServerURL+"/api/auth/sign-in", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("sign-in failed: %s — %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Token, nil
}

func (api *APIClient) SignUp(email, name, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"name":     name,
		"password": password,
	})

	resp, err := api.HTTPClient.Post(api.ServerURL+"/api/auth/sign-up", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("sign-up failed: %s — %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Token, nil
}

func (api *APIClient) RegisterAgent(jwt, name, machineID string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"name":       name,
		"machine_id": machineID,
	})

	req, err := http.NewRequest("POST", api.ServerURL+"/api/agent", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("register agent failed: %s — %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		Agent struct {
			Token string `json:"token"`
		} `json:"agent"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Agent.Token, nil
}

func (api *APIClient) ListAgents(jwt string) ([]models.Agent, error) {

	req, err := http.NewRequest("GET", api.ServerURL+"/api/agents", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list agents failed: %s — %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		Agents []models.Agent
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Agents, nil
}
