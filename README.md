# MyDeploy

```
       .
      ":"                 __  ___     ___           __
    ___:____     |"\/"|  /  |/  /_ __/ _ \___ ___  / /__  __ __
  ,'        `.    \  /  / /|_/ / // / // / -_) _ \/ / _ \/ // /
  |  O        \___/  | /_/  /_/\_, /____/\__/ .__/_/\___/\_, /
~^~^~^~^~^~^~^~^~^~^~^~       /___/        /_/          /___/
```

Self-hosted deployment platform. Deploy Docker containers to remote machines through a microservice backend and interactive TUI client.

## Architecture

The system consists of several microservices, each responsible for a specific domain. Communication is handled via HTTP (public API) and gRPC (internal service-to-service).

- **Gateway**: Single entry point, JWT validation, and request routing.
- **Auth Service**: User management, registration, and authentication.
- **Agent Service**: Manages connected agents via WebSockets.
- **Deploy Service**: Handles deployment logic and communicates with agents.
- **Template Service**: Provides application templates (Postgres, Redis, etc.).

## Development & Deployment

### Prerequisites

- **Go** 1.24+
- **Docker** & **Docker Compose**
- **Kubernetes** (Minikube recommended for local testing)
- **Helm** (for K8s deployment)

### Local Development (Docker Compose)

The easiest way to start all services and databases:

```bash
docker compose up --build
```

The Gateway will be available at `http://localhost:8080`.

### Kubernetes Deployment (Minikube)

To deploy the full stack into a local Kubernetes cluster:

1. **Start Minikube & Enable Ingress:**
   ```bash
   minikube start
   minikube addons enable ingress
   ```

2. **Build and Load Images:**
   ```bash
   make docker-build-all
   minikube image load my-registry/auth-service:v1.0.2
   minikube image load my-registry/agent-service:v1.0.2
   minikube image load my-registry/deploy-service:v1.0.2
   minikube image load my-registry/template-service:v1.0.2
   minikube image load my-registry/gateway-service:v1.0.2
   ```

3. **Install via Helm:**
   ```bash
   make k8s-install
   ```

4. **Access the API:**
   Run `minikube tunnel` in a separate terminal and add the following to your `hosts` file:
   ```text
   127.0.0.1 api.my-deploy.local
   ```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make build` | Build all Go binaries to `bin/` |
| `make docker-build-all` | Build all Docker images (v1.0.2) |
| `make k8s-install` | Install/Setup everything in Kubernetes |
| `make k8s-status` | Check status of Pods, Services, and Ingress |
| `make k8s-restart` | Restart all deployments to apply changes |
| `make k8s-uninstall` | Remove all resources from the cluster |

## Database Migrations

The system features an automatic migration system. Upon startup, each service checks its respective migration directory and applies any missing `.sql` files:

- **Auth**: `migrations/auth/` -> `auth_db`
- **Agent**: `migrations/agent/` -> `agent_db`
- **Deploy**: `migrations/deploy/` -> `deploy_db`

## API Testing

You can test the API using `curl` or Postman. 

**Health Check:**
```bash
curl -i http://api.my-deploy.local/health
```

**Registration (Sign-Up):**
```bash
curl -X POST http://api.my-deploy.local/api/auth/sign-up \
     -H "Content-Type: application/json" \
     -d '{"email":"user@example.com", "password":"securepassword"}'
```

## Tech Stack

- **Backend**: Go (stdlib `net/http`, `gRPC`, `WebSockets`)
- **Database**: PostgreSQL 16
- **Orchestration**: Kubernetes (Helm) / Docker Compose
- **CLI**: Bubble Tea (TUI)
- **Security**: JWT Authentication
