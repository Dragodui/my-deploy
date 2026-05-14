package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type installConfigRequest struct {
	Token     string `json:"token"`
	MachineID string `json:"machine_id"`
	ServerURL string `json:"server_url,omitempty"`
	BinaryURL string `json:"binary_url,omitempty"`
}

type bootstrapExchangeResponse struct {
	Agent struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Token     string `json:"token"`
		MachineID string `json:"machine_id"`
	} `json:"agent"`
}

type localAgentConfig struct {
	AgentName  string `json:"agent_name"`
	AgentID    string `json:"agent_id"`
	URL        string `json:"url"`
	AgentToken string `json:"agent_token"`
	MachineID  string `json:"machine_id"`
	AgentMode  string `json:"agent_mode"`
}

type installMetaResponse struct {
	PublicURL        string `json:"public_url"`
	DefaultBinaryURL string `json:"default_binary_url"`
}

func InstallMetaHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(installMetaResponse{
			PublicURL:        strings.TrimRight(cfg.PublicURL, "/"),
			DefaultBinaryURL: strings.TrimSpace(cfg.AgentBinaryURL),
		})
	}
}

func InstallScriptHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		serverURL := strings.TrimSpace(r.URL.Query().Get("server_url"))
		binaryURL := r.URL.Query().Get("binary_url")

		if token == "" {
			http.Error(w, "token is required", http.StatusBadRequest)
			return
		}
		if serverURL == "" {
			serverURL = strings.TrimRight(cfg.PublicURL, "/")
		}
		if binaryURL == "" {
			binaryURL = strings.TrimSpace(cfg.AgentBinaryURL)
		}
		if serverURL == "" || binaryURL == "" {
			http.Error(w, "server install defaults are not configured", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
		io.WriteString(w, renderInstallScript(serverURL, token, binaryURL))
	}
}

func InstallConfigHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req installConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Token == "" || req.MachineID == "" {
			http.Error(w, "token and machine_id are required", http.StatusBadRequest)
			return
		}

		serverURL := strings.TrimRight(cfg.PublicURL, "/")
		if strings.TrimSpace(req.ServerURL) != "" {
			serverURL = strings.TrimRight(strings.TrimSpace(req.ServerURL), "/")
		}

		upstreamBody, _ := json.Marshal(struct {
			Token     string `json:"token"`
			MachineID string `json:"machine_id"`
		}{
			Token:     req.Token,
			MachineID: req.MachineID,
		})

		upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, cfg.AgentURL.String()+"/api/agent/bootstrap/exchange", bytes.NewReader(upstreamBody))
		if err != nil {
			http.Error(w, "failed to prepare upstream request", http.StatusInternalServerError)
			return
		}
		upstreamReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(upstreamReq)
		if err != nil {
			http.Error(w, "failed to contact agent service", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			http.Error(w, strings.TrimSpace(string(body)), resp.StatusCode)
			return
		}

		var exchange bootstrapExchangeResponse
		if err := json.NewDecoder(resp.Body).Decode(&exchange); err != nil {
			http.Error(w, "failed to decode agent response", http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(localAgentConfig{
			AgentName:  exchange.Agent.Name,
			AgentID:    exchange.Agent.ID,
			URL:        serverURL,
			AgentToken: exchange.Agent.Token,
			MachineID:  exchange.Agent.MachineID,
			AgentMode:  "remote",
		})
	}
}

func renderInstallScript(serverURL, token, binaryURL string) string {
	return fmt.Sprintf(`#!/bin/sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "Run this installer as root." >&2
  exit 1
fi

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

require_cmd curl
require_cmd install
require_cmd systemctl

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

MACHINE_ID="$(cat /etc/machine-id 2>/dev/null || hostname)"
CONFIG_DIR="/root/.mydeploy"
CONFIG_PATH="$CONFIG_DIR/config.json"
BIN_PATH="/usr/local/bin/mydeploy-agent"
SERVICE_PATH="/etc/systemd/system/mydeploy-agent.service"

curl -fsSL %s -o "$TMP_DIR/mydeploy-agent"
install -m 0755 "$TMP_DIR/mydeploy-agent" "$BIN_PATH"

mkdir -p "$CONFIG_DIR"
curl -fsSL -X POST %s \
  -H 'Content-Type: application/json' \
  -d "{\"token\":\"%s\",\"machine_id\":\"$MACHINE_ID\",\"server_url\":\"%s\"}" \
  -o "$CONFIG_PATH"
chmod 600 "$CONFIG_PATH"

cat > "$SERVICE_PATH" <<'EOF'
[Unit]
Description=MyDeploy Agent
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/mydeploy-agent
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now mydeploy-agent
systemctl status mydeploy-agent --no-pager
`, shellSingleQuote(binaryURL), shellSingleQuote(strings.TrimRight(serverURL, "/")+"/api/install/agent/config"), escapeJSON(token), escapeJSON(strings.TrimRight(serverURL, "/")))
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func escapeJSON(s string) string {
	b, _ := json.Marshal(s)
	return strings.Trim(string(b), `"`)
}
