# MyDeploy

```
       .
      ":"                 __  ___     ___           __
    ___:____     |"\/"|  /  |/  /_ __/ _ \___ ___  / /__  __ __
  ,'        `.    \  /  / /|_/ / // / // / -_) _ \/ / _ \/ // /
  |  O        \___/  | /_/  /_/\_, /____/\__/ .__/_/\___/\_, /
~^~^~^~^~^~^~^~^~^~^~^~       /___/        /_/          /___/
```

Self-hosted deployment platform. Deploy Docker containers to remote machines through a microservice backend and interactive TUI client. Includes a Wails (React + TS) desktop app scaffold in `desktop/`.

## Architecture

```
                          ┌──────────┐
                          │ Gateway  │ :8080
                          └────┬─────┘
               ┌──────────┬────┼────┬──────────┐
               ▼          ▼    ▼    ▼          ▼
           ┌───────┐  ┌──────┐ ┌──────┐   ┌──────────┐
           │ Auth  │  │Agent │ │Deploy│   │ Template │
           │  Svc  │  │ Svc  │ │ Svc  │   │  Svc     │
           └──┬────┘  └──┬───┘ └──┬───┘   └──────────┘
              │  gRPC   │ gRPC    │HTTP
              ▼         │         ▼
         [auth_db]      │     [deploy_db]
                        ▼
                   [agent_db]
                        │
                     WebSocket
                        │
                     [Agent]
```

### Services

| Service | Entry point | Port | DB | Role |
|---------|------------|------|-----|------|
| **gateway** | `cmd/gateway/main.go` | 8080 | — | Reverse proxy, JWT validation, routes to services |
| **auth-service** | `cmd/auth-service/main.go` | 8081 (HTTP) / 9081 (gRPC) | `auth_db` | Registration, login, JWT tokens |
| **agent-service** | `cmd/agent-service/main.go` | 8082 (HTTP) / 9082 (gRPC) | `agent_db` | Agent registry, WebSocket hub |
| **deploy-service** | `cmd/deploy-service/main.go` | 8083 | `deploy_db` | Deployment CRUD, sends commands via gRPC |
| **template-service** | `cmd/template-service/main.go` | 8084 | — | YAML app templates from `templates/` |
| **agent** | `cmd/agent/main.go` | — | — | Runs on target machine, executes Docker commands |
| **CLI** | `cmd/cli/main.go` | — | — | Interactive TUI (Bubble Tea) |
| **desktop** | `desktop/` | — | — | Wails desktop app (React + TS) |

### Inter-service communication

- **Gateway -> services** — HTTP reverse proxy. Validates JWT for user routes, injects `X-User-ID` header.
- **deploy-service -> agent-service** — gRPC (`agentpb`): send deploy/start/stop commands, get progress.
- **deploy-service -> template-service** — HTTP (`/internal/templates/{id}`): resolve template details.
- **auth-service** — gRPC (`authpb`): `ValidateUser` for internal user lookups.
- **agent-service <-> agent** — WebSocket (`/ws/agent`): bidirectional JSON messages.

## Getting Started

### Prerequisites

- Go 1.24+
- Docker (on agent machines)
- PostgreSQL (or use Docker Compose)
- Node.js 18+ and Wails CLI (only for the desktop app)

### Docker Compose (recommended)

```bash
docker compose up --build
```

This starts all services, three PostgreSQL instances, and the gateway on port 8080.

### Run services individually

```bash
# Gateway
PORT=8080 JWT_SECRET=secret AUTH_SERVICE_URL=http://localhost:8081 \
  AGENT_SERVICE_URL=http://localhost:8082 DEPLOY_SERVICE_URL=http://localhost:8083 \
  TEMPLATE_SERVICE_URL=http://localhost:8084 go run cmd/gateway/main.go

# Auth service
PORT=8081 GRPC_PORT=9081 JWT_SECRET=secret \
  DB_DSN="postgres://auth:authpass@localhost:5433/auth_db?sslmode=disable" \
  go run cmd/auth-service/main.go

# Agent service
PORT=8082 GRPC_PORT=9082 \
  DB_DSN="postgres://agent:agentpass@localhost:5434/agent_db?sslmode=disable" \
  go run cmd/agent-service/main.go

# Deploy service
PORT=8083 AGENT_URL=localhost:9082 TEMPLATE_URL=http://localhost:8084 \
  DB_DSN="postgres://deploy:deploypass@localhost:5435/deploy_db?sslmode=disable" \
  go run cmd/deploy-service/main.go

# Template service
PORT=8084 TEMPLATES_DIR=./templates go run cmd/template-service/main.go
```

### CLI

```bash
go run cmd/cli/main.go
```

The TUI guides you through registration/login, agent setup, and deployment management.

### Desktop App (Wails)

```bash
cd desktop
wails dev
```

```bash
cd desktop
wails build
```

### Agent

**Local mode** — managed by CLI as a daemon (start/stop from TUI home menu).

**Remote mode:**

```bash
go run cmd/agent/main.go --url http://your-server:8080 --token <agent-token>
```

Agent config is stored in `~/.mydeploy/config.json`.

### Build

```bash
make build          # build all binaries to bin/
make build-cli      # bin/mydeploy
make build-agent    # bin/mydeploy-agent
```

## API

All endpoints go through the gateway on `:8080`.

### Auth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/sign-up` | — | Register (`email`, `name`, `password`) |
| POST | `/api/auth/sign-in` | — | Login (`email`, `password`) |
| GET | `/api/me` | JWT | Current user profile |

### Agents

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/agent` | JWT | Register or get agent |
| GET | `/api/agents` | JWT | List user's agents |
| GET | `/ws/agent` | Agent token | Agent WebSocket connection (`X-Agent-Token`) |

### Deployments

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/deployments` | JWT | Create deployment |
| GET | `/api/deployments?agent_id=` | JWT | List deployments |
| GET | `/api/deployments/{id}` | JWT | Get deployment + status |
| DELETE | `/api/deployments/{id}` | JWT | Delete deployment |
| POST | `/api/deployments/{id}/start` | JWT | Start container |
| POST | `/api/deployments/{id}/stop` | JWT | Stop container |
| PATCH | `/api/deployments/{id}` | JWT | Update deployment fields |

Gateway also exposes `GET /ws/logs/{deploy_id}` as a WebSocket proxy to deploy-service; a deploy-service handler for this is not implemented yet.

### Templates

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/templates` | JWT | List app templates |

### gRPC (internal)

**AuthInternal** (`:9081`):
- `ValidateUser` — verify user by ID

**AgentInternal** (`:9082`):
- `IsConnected` — check if agent is online
- `SendCommand` — send deploy/start/stop to agent
- `GetAgent` — get agent details
- `StreamLogs` — stream container logs
- `GetProgress` — get deployment progress

## Project Structure

```
cmd/
  gateway/             API gateway
  auth-service/        auth service
  agent-service/       agent service
  deploy-service/      deploy service
  template-service/    template service
  agent/               agent binary
  cli/                 CLI binary
internal/
  gateway/             reverse proxy, routing, JWT middleware
  authSvc/             auth: config, repository, service, handler
  agentSvc/            agent: config, repository, service, handler, WebSocket hub
  deploySvc/           deploy: config, repository, service, handler
  templateSvc/         template: registry, service, handler
  agent/               agent client: WebSocket, Docker operations, config
  cli/                 TUI screens (Bubble Tea)
  daemon/              agent daemon management
  shared/
    models/            domain models
    auth/              JWT utilities
    middleware/        HTTP middleware
    proto/             protobuf definitions (authpb, agentpb)
proto/                 .proto source files
migrations/
  auth/                auth_db migrations
  agent/               agent_db migrations
  deploy/              deploy_db migrations
templates/             YAML app templates
desktop/               Wails desktop app (React + TS)
```

## Tech Stack

- **Go** 1.24, stdlib `net/http`
- **PostgreSQL** 16 (`lib/pq`)
- **gRPC** + Protobuf for inter-service communication
- **WebSocket** (`gorilla/websocket`) for agent connections
- **JWT** (`golang-jwt/jwt`)
- **CLI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **Docker SDK** (`moby/moby/client`) on agents
- **Docker Compose** for local development
- **Desktop UI**: Wails v2 + React + Vite (TypeScript)

## TODO

- [ ] Web dashboard
- [ ] Desktop app features (Wails UI + API integration)
- [ ] Deployment settings editing from CLI
- [ ] Agent health monitoring and auto-reconnect UI
- [ ] More app templates (advanced stacks and multi-container setups)
- [ ] Environment variables management per agent
- [ ] Multi-user access control (teams, roles)
- [ ] HTTPS / TLS support
- [ ] CI/CD integration (deploy on git push)
- [ ] Prometheus metrics + Grafana dashboards
- [ ] Resource usage monitoring
- [ ] Container volume management
- [ ] Notifications (Telegram, Discord, webhooks)
