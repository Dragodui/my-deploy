# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
# Run individual services locally
go run cmd/auth-service/main.go
go run cmd/agent-service/main.go
go run cmd/deploy-service/main.go
go run cmd/template-service/main.go
go run cmd/gateway/main.go
go run cmd/cli/main.go
go run cmd/agent/main.go --token <token> --url <server-url>

# Run all services with Docker Compose
docker compose up --build

# Generate protobuf (auth and agent services use gRPC for inter-service communication)
# Proto files: internal/shared/proto/authpb/proto/, internal/shared/proto/agentpb/proto/
```

There are no tests in this codebase yet.

## Architecture

MyDeploy is a **microservice-based deployment platform** that deploys Docker containers to remote machines.

### Services (6 binaries)

| Service | Entry point | Port | DB | Purpose |
|---------|------------|------|-----|---------|
| **gateway** | `cmd/gateway/main.go` | 8080 | - | Reverse proxy, JWT validation, routes to downstream services |
| **auth-service** | `cmd/auth-service/main.go` | 8081 (HTTP), 9081 (gRPC) | `auth_db` | User registration, login, JWT |
| **agent-service** | `cmd/agent-service/main.go` | 8082 (HTTP), 9082 (gRPC) | `agent_db` | Agent registration, WebSocket hub for connected agents |
| **deploy-service** | `cmd/deploy-service/main.go` | 8083 | `deploy_db` | Deployment CRUD, sends commands to agents via agent-service gRPC |
| **template-service** | `cmd/template-service/main.go` | 8084 | - | Serves YAML app templates from `templates/` dir |
| **agent** | `cmd/agent/main.go` | - | - | Runs on target machine, connects via WebSocket, executes Docker commands |
| **CLI** | `cmd/cli/main.go` | - | - | Interactive TUI (Bubble Tea) |

### Inter-service communication

- **Gateway → services**: HTTP reverse proxy. Gateway validates JWT and forwards `X-User-ID` header.
- **deploy-service → agent-service**: gRPC (`agentpb`) to send deploy/start/stop commands to connected agents.
- **deploy-service → template-service**: HTTP (`/internal/templates/{id}`) to resolve template details.
- **auth-service** exposes gRPC (`authpb`) for internal user lookups.
- **agent-service ↔ agent**: WebSocket (`/ws/agent`) with JSON messages for deploy, start, stop, status, logs.

### Code layout

Each service follows the same pattern: `config.go` → `repository.go` → `service.go` → `handler.go`, all in a single package under `internal/<serviceName>Svc/`.

Shared code lives in `internal/shared/`: models, auth utilities, HTTP middleware, and protobuf definitions.

The agent client code (`internal/agent/`) handles WebSocket connection, Docker operations, and local config (`~/.mydeploy/config.json`).

The CLI (`internal/cli/`) is a set of Bubble Tea models: auth → agent selection → home → deploy/deploy_list/logs screens.

### Environment variables

Each service reads config from env vars. Key vars per service:
- All DB services: `DB_DSN`, `PORT`
- auth/agent services: `GRPC_PORT`, `JWT_SECRET`
- gateway: `JWT_SECRET`, `AUTH_SERVICE_URL`, `AGENT_SERVICE_URL`, `DEPLOY_SERVICE_URL`, `TEMPLATE_SERVICE_URL`
- deploy-service: `AGENT_URL` (gRPC), `TEMPLATE_URL` (HTTP)
- template-service: `TEMPLATES_DIR`

### Database

Each DB service has its own PostgreSQL database. Migrations are in `migrations/{auth,agent,deploy}/`. Each service has a separate Postgres instance in docker-compose.
