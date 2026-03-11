package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

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
