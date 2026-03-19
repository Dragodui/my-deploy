# MyDeploy

```
       .
      ":"                 __  ___     ___           __
    ___:____     |"\/"|  /  |/  /_ __/ _ \___ ___  / /__  __ __
  ,'        `.    \  /  / /|_/ / // / // / -_) _ \/ / _ \/ // /
  |  O        \___/  | /_/  /_/\_, /____/\__/ .__/_/\___/\_, /
~^~^~^~^~^~^~^~^~^~^~^~       /___/        /_/          /___/
```

A self-hosted deployment platform. Deploy Docker containers to remote machines through a central server using a beautiful TUI client.

## Architecture

```
[Server]      central API + PostgreSQL
   |                  |
 REST             WebSocket
   |                  |
[CLI]               [Agent]
 TUI        daemon on target machine
```

**Server** (`cmd/main.go`) — REST API, manages users, agents, and deployments. Stores state in PostgreSQL. Logs to stdout and `logs/server.log`.

**Agent** (`cmd/agent/main.go`) — runs on the target machine with Docker. Connects to the server via WebSocket, receives deploy commands, and manages containers. Supports local (daemon) and remote modes.

**CLI** (`cmd/cli/main.go`) — interactive TUI client (built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)). Register, log in, manage agents, create deployments from templates or custom images, and view deployment status.

## Getting Started

### Prerequisites

- Go 1.24+
- PostgreSQL
- Docker (on agent machines)

### Server

```bash
# Set environment variables (DB connection, JWT secret, port)
export DB_DSN="postgres://user:pass@localhost:5432/mydeploy?sslmode=disable"
export JWT_SECRET="your-secret"
export PORT=8080

# Run the server
go run cmd/main.go
```

Or with Docker Compose:

```bash
# Configure .env with DB_DSN, JWT_SECRET, PORT
docker compose up --build
```

Migrations run automatically on startup from the `migrations/` directory.

### CLI

```bash
go run cmd/cli/main.go
```

On first launch the CLI will guide you through:

1. **Registration / Login** — create an account or sign in
2. **Agent setup** — select an existing agent or create a new one (with optional Docker host)
3. **Home screen** — main menu:
   - **Deploy** — create a deployment from a template or custom Docker image, with real-time progress
   - **Deploy list** — view all deployments with live Docker status, start/stop/delete containers, view live logs
   - **Start / Stop agent** — manage the local agent daemon
   - **Change agent** — switch to a different agent
   - **Logout**

Config is saved to `~/.mydeploy/config.json`.

### Agent

**Local mode** (managed by CLI as a daemon):

The CLI automatically starts the agent daemon after creating an agent. You can also start/stop it from the home menu. Daemon PID and logs are stored in `~/.mydeploy/`.

**Remote mode** (standalone on a remote server):

```bash
go run cmd/agent/main.go --url http://your-server:8080 --token <agent-token>
```

The agent token is displayed after agent creation in the CLI. On subsequent runs, the agent reads its config from `~/.mydeploy/config.json`.

## App Templates

Templates define pre-configured deployments as YAML files in `internal/templates/`. Each template specifies an image, ports, volumes, environment variables, and resource limits.

Available templates:
- **Minecraft Server** — Vanilla Minecraft Java server (`itzg/minecraft-server`) — 2G RAM, 1 CPU
- **Nginx** — Simple web server (`nginx:alpine`)

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | - | Health check |
| POST | `/api/auth/sign-up` | - | Register |
| POST | `/api/auth/sign-in` | - | Login |
| GET | `/api/me` | JWT | Current user info |
| POST | `/api/agent` | JWT | Register or get agent |
| GET | `/api/agents` | JWT | List user's agents |
| GET | `/api/templates` | JWT | List available app templates |
| POST | `/api/deployments` | JWT | Create deployment |
| GET | `/api/deployments?agent_id=` | JWT | List deployments |
| GET | `/api/deployments/{id}` | JWT | Get deployment |
| DELETE | `/api/deployments/{id}` | JWT | Delete deployment |
| POST | `/api/deployments/{id}/start` | JWT | Start container |
| POST | `/api/deployments/{id}/stop` | JWT | Stop container |
| GET | `/ws/agent` | Agent Token | Agent WebSocket |
| GET | `/ws/logs/{id}` | JWT | Live container logs (WebSocket) |

## Project Structure

```
cmd/
  main.go              server entry point
  cli/main.go          CLI entry point
  agent/main.go        agent entry point
  server/server.go     HTTP router & dependency wiring
internal/
  agent/               agent client, config, WebSocket handler, API client
  auth/                JWT generation, password hashing
  cli/                 TUI screens (login, register, agent, home, deploy, deploy list, logs)
  config/              server config (env-based)
  daemon/              agent daemon management (start, stop, status, PID tracking)
  db/                  database connection & auto-migrations
  http/                WebSocket handler
  http/handler/        HTTP handlers (auth, agent, deploy, templates)
  http/middleware/     JWT & agent token middleware
  models/              domain models (User, Agent, Deployment, AppTemplate)
  registry/            in-memory agent WebSocket connection registry
  repository/          database queries
  service/             business logic (auth, agent, deploy, templates)
  templates/           app template definitions (YAML)
migrations/            SQL migration files (auto-applied on startup)
```

## Tech Stack

- **Server**: Go stdlib `net/http`, PostgreSQL (`lib/pq`), `golang-jwt/jwt`, `gorilla/websocket`
- **CLI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **Agent**: Docker SDK (`moby/moby/client`), `gorilla/websocket`
- **Config**: `goccy/go-yaml`, `joho/godotenv`

## TODO

### Done

- [x] User registration and JWT authentication
- [x] Agent registration and WebSocket connection
- [x] User registration and JWT authentication
- [x] Agent registration and WebSocket connection
- [x] Deploy from custom Docker image (name, image, ports, env)
- [x] Deploy from YAML app templates (Minecraft, Nginx)
- [x] Resource limits (memory, CPU) from templates and user overrides
- [x] Async deployment with real-time progress (pulling, creating, starting)
- [x] Deployment list with live Docker status (via container inspect)
- [x] Start / stop / delete containers from TUI
- [x] Live container logs streaming via WebSocket (saved to `~/.mydeploy/logs/`)
- [x] Agent daemon management (start/stop from TUI)
- [x] Local and remote agent modes
- [x] Interactive TUI with Bubble Tea
- [x] Auto-migrations on server startup
- [x] Docker Compose support
- [x] Server error logging to file (`logs/server.log`)

### Planned

- [ ] Microservice architecture (split server into auth, deploy, agent gateway services)
- [ ] Desktop app (Electron / Tauri / Wails)
- [ ] Web dashboard as an alternative to TUI
- [ ] Deployment settings editing from CLI
- [ ] Agent health monitoring and auto-reconnect status in UI
- [ ] More app templates (PostgreSQL, Redis, Node.js)
- [ ] Environment variables management per agent
- [ ] Multi-user access control (teams, roles)
- [ ] HTTPS / TLS support for server and agent connections
- [ ] CI/CD integration (deploy on git push)
- [ ] Prometheus metrics + Grafana dashboards (per-service /metrics endpoint, request rate, latency, error rate)
- [ ] Resource usage monitoring (CPU, memory, disk)
- [ ] Container volume management in CLI
- [ ] Notifications (Telegram, Discord, webhooks)
