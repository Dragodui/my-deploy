# MyDeploy

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

**Server** (`cmd/main.go`) — REST API, manages users, agents, and deployments. Stores state in PostgreSQL.

**Agent** (`cmd/agent/main.go`) — runs on the target machine with Docker. Connects to the server via WebSocket, receives deploy commands, and manages containers.

**CLI** (`cmd/cli/main.go`) — interactive TUI client (built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)). Register, log in, select an agent, and create deployments.

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
cd my-deploy
go run cmd/main.go
```

Migrations run automatically on startup from the `migrations/` directory.

### CLI

```bash
go run cmd/cli/main.go
```

On first launch the CLI will guide you through:

1. **Registration / Login** — create an account or sign in
2. **Agent setup** — select an existing agent or create a new one
3. **Home screen** — main menu with deploy options

Config is saved to `~/.mydeploy/config.json`.

### Agent

```bash
go run cmd/agent/main.go
```

On first launch the agent runs an interactive setup (email, password, agent name, Docker host), then connects to the server via WebSocket and waits for commands.

Config is saved to `~/.mydeploy/config.json`.

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/sign-up` | - | Register |
| POST | `/api/auth/sign-in` | - | Login |
| GET | `/api/me` | JWT | Current user info |
| POST | `/api/agent` | JWT | Register or get agent |
| GET | `/api/agents` | JWT | List user's agents |
| POST | `/api/deployments` | JWT | Create deployment |
| GET | `/api/deployments?agent_id=` | JWT | List deployments |
| GET | `/api/deployments/{id}` | JWT | Get deployment |
| DELETE | `/api/deployments/{id}` | JWT | Delete deployment |
| GET | `/ws/agent` | Agent Token | Agent WebSocket |

## Project Structure

```
cmd/
  main.go              server entry point
  cli/main.go          CLI entry point
  agent/main.go        agent entry point
  server/server.go     HTTP router & dependency wiring
internal/
  agent/               agent client, config, WebSocket, setup
  auth/                JWT generation, password hashing
  cli/                 TUI screens (login, register, agent, home)
  config/              server config
  db/                  database connection & migrations
  http/                WebSocket handler
  http/handler/        HTTP handlers (auth, agent, deploy)
  http/middleware/      JWT & agent token middleware
  models/              domain models (User, Agent, Deployment)
  registry/            in-memory agent connection registry
  repository/          database queries
  service/             business logic
  templates/           app templates (YAML-based)
```

## Tech Stack

- **Server**: Go stdlib `net/http`, PostgreSQL, `golang-jwt/jwt`, `gorilla/websocket`
- **CLI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **Agent**: Docker SDK (`moby/moby/client`), `gorilla/websocket`
