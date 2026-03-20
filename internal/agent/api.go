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

	"github.com/dragodui/my-deploy/internal/shared/models"
	"github.com/gorilla/websocket"
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

func (api *APIClient) SignIn(email, password string) (string, string, error) {
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})

	resp, err := api.HTTPClient.Post(api.ServerURL+"/api/auth/sign-in", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("sign-in failed: %s — %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		Token string `json:"token"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	return result.Token, result.Name, nil
}

func (api *APIClient) SignUp(email, name, password string) (string, string, error) {
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"name":     name,
		"password": password,
	})

	resp, err := api.HTTPClient.Post(api.ServerURL+"/api/auth/sign-up", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("sign-up failed: %s — %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var result struct {
		Token string `json:"token"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	return result.Token, result.Name, nil
}

func (api *APIClient) RegisterAgent(jwt, name, machineID string) (string, string, error) {
	body, _ := json.Marshal(map[string]string{
		"name":       name,
		"machine_id": machineID,
	})

	req, err := http.NewRequest("POST", api.ServerURL+"/api/agent", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("register agent failed: %s — %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		Agent struct {
			ID    string `json:"id"`
			Token string `json:"token"`
		} `json:"agent"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	return result.Agent.ID, result.Agent.Token, nil
}

type MeResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (api *APIClient) Me(jwt string) (*MeResponse, error) {
	req, err := http.NewRequest("GET", api.ServerURL+"/api/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("me failed: %s — %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var result MeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
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

func (api *APIClient) ListTemplates(jwt string) ([]models.AppTemplate, error) {
	req, err := http.NewRequest("GET", api.ServerURL+"/api/templates", nil)
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
		return nil, fmt.Errorf("list templates failed: %s — %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		Templates []models.AppTemplate
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Templates, nil
}

func (api *APIClient) CreateDeployment(jwt, agentID string, deploymentReq models.DeployRequest) (*models.Deployment, error) {
	type body struct {
		AgentID string `json:"agent_id"`
		models.DeployRequest
	}

	data, err := json.Marshal(body{AgentID: agentID, DeployRequest: deploymentReq})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", api.ServerURL+"/api/deployments", bytes.NewReader(data))
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create deployment failed: %s — %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		Deployment models.Deployment `json:"deployment"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Deployment, nil
}

func (api *APIClient) GetDeployment(jwt, id string) (*models.Deployment, error) {
	req, err := http.NewRequest("GET", api.ServerURL+"/api/deployments/"+id, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+jwt)

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get deployment failed: %s — %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var result models.Deployment
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (api *APIClient) ListDeployments(jwt, agentID string) ([]models.Deployment, error) {
	req, err := http.NewRequest("GET", api.ServerURL+"/api/deployments?agent_id="+agentID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list deployments failed: %s — %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var result []models.Deployment
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (api *APIClient) StopDeployment(jwt, deployID string) error {
	return api.manageDeployment(jwt, deployID, "stop")
}

func (api *APIClient) StartDeployment(jwt, deployID string) error {
	return api.manageDeployment(jwt, deployID, "start")
}

func (api *APIClient) DeleteDeployment(jwt, deployID string) error {
	req, err := http.NewRequest("DELETE", api.ServerURL+"/api/deployments/"+deployID, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+jwt)
	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete deployment failed: %s — %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	return nil
}

func (api *APIClient) manageDeployment(jwt, deployID, action string) error {
	if action != "start" && action != "stop" {
		return fmt.Errorf("action is not correct, only start and stop allowed")
	}

	req, err := http.NewRequest("POST", api.ServerURL+"/api/deployments/"+deployID+"/"+action, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+jwt)

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s deployment failed: %s — %s", action, resp.Status, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func (api *APIClient) ConnectLogs(jwt, deployID string) (*websocket.Conn, error) {
	wsURL := strings.Replace(api.ServerURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = wsURL + "/ws/logs/" + deployID

	header := make(http.Header)
	header.Set("Authorization", "Bearer "+jwt)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return nil, fmt.Errorf("connect logs failed: %w", err)
	}

	return conn, nil
}
