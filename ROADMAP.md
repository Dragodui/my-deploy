# Roadmap: Self-Service Agent Onboarding

## Context

Currently the agent is started manually with `--server` and `--token` flags. There is no auth, no user model, and no token validation. The goal is: user installs CLI → logs in → agent auto-registers and starts working. Each user owns their own agents (multi-tenancy).

### Target UX

```bash
$ mydeploy setup --server https://deploy.example.com
Email: user@example.com
Password: ****
✓ Agent registered. Starting...

$ mydeploy run   # reads token from ~/.mydeploy/config.json
```

---

## Phase 1: Database + Models

### Migrations

Create SQL files in `migrations/`:

**001_users.sql**
```sql
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT UNIQUE NOT NULL,
    password    TEXT NOT NULL,  -- bcrypt hash
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**002_agents.sql**
```sql
CREATE TABLE agents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    token       TEXT UNIQUE NOT NULL,  -- crypto/rand 32-byte hex
    name        TEXT NOT NULL,         -- hostname
    machine_id  TEXT NOT NULL,         -- machine fingerprint
    last_seen   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, machine_id)
);
CREATE INDEX idx_agents_token ON agents(token);
```

**003_deployments.sql**
```sql
CREATE TABLE deployments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id      UUID NOT NULL REFERENCES agents(id),
    name          TEXT NOT NULL,
    app_id        TEXT,
    image         TEXT NOT NULL,
    container_id  TEXT,
    ports         JSONB DEFAULT '[]',
    volumes       JSONB DEFAULT '[]',
    env           JSONB DEFAULT '[]',
    status        TEXT NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Models

- [x] Create `internal/models/user.go` — User struct
- [x] Create `internal/models/agent_record.go` — AgentRecord struct (persistent agent model)
- [x] Update `internal/models/deploy.go` — add `AgentID` to Deployment
- [x] Remove duplicate `Command` from `internal/models/agent.go` (already defined in `internal/agent/messages.go`)

### DB

- [x] Add `RunMigrations()` to `internal/db/db.go`

---

## Phase 2: Auth

### Dependencies

```bash
go get golang.org/x/crypto/bcrypt
go get github.com/golang-jwt/jwt/v5
```

### Tasks

- [x] `internal/config/config.go` — add `JWTSecret` from `JWT_SECRET` env var
- [x] `internal/auth/auth.go` — GenerateToken, ValidateToken, HashPassword, CheckPassword
- [x] `internal/repository/user.go` — Create, GetByEmail, GetByID
- [x] `internal/service/auth.go` — Signup(email, password) → JWT, Login(email, password) → JWT
- [x] `internal/http/handler/auth.go` — `POST /api/auth/sign-up`, `POST /api/auth/sign-in`
- [x] `internal/http/middleware/jwt.go` — JWT middleware, extracts userID into context

---

## Phase 3: Agent Registration API

- [x] `internal/repository/agent.go` — Create, GetByToken, GetByUserAndMachine, ListByUser, UpdateLastSeen
- [x] `internal/service/agent.go` — RegisterOrGet(userID, name, machineID) — idempotent registration
- [x] `internal/http/handler/agent.go` — `POST /api/agent` (JWT required)

**RegisterOrGet** — key logic: if an agent already exists for user+machine, return existing token. Otherwise create a new one. This allows running setup repeatedly without duplication.

---

## Phase 4: WebSocket Auth Hardening

- [x] `internal/http/middleware/agent.go` — AgentAuth middleware, validates token via `AgentRepository.GetByToken()`; puts `agentID` in context
- [x] `internal/http/ws.go` — register/unregister by `agentID` (from middleware context)
- [x] `internal/registry/registry.go` — map key: `agentID` (UUID)
- [x] `internal/service/deploy.go` — `Create` accepts `agentID`, finds agent in registry by ID
- [x] `internal/http/handler/deploy.go` — accepts `agent_id` in request body

---

## Phase 5: Agent CLI

- [x] `internal/agent/machine.go` — GenerateMachineID via uuid
- [x] `internal/agent/localconfig.go` — `~/.mydeploy/config.json`: Load/Save
- [x] `internal/agent/api.go` — APIClient: SignIn, SignUp, RegisterAgent
- [x] `internal/agent/setup.go` — interactive setup: prompt email/password → auth API → register agent → save config
- [x] `internal/agent/client.go` — X-Agent-Token header instead of Bearer JWT
- [x] `cmd/agent/main.go` — Load config → setup if missing → run agent
- [ ] `cmd/agent/main.go` — CLI subcommands: setup, agent, deploy, list, stop

---

## Phase 6: Wiring

- [x] `cmd/server/server.go` — init new repos/services/handlers, register routes
- [x] `internal/repository/deploy.go` — real SQL queries instead of mock
- [x] `internal/http/handler/deploy.go` — CRUD endpoints for deployments

---

## Key Architecture Decisions

| Decision | Rationale |
|----------|-----------|
| Agent token ≠ JWT | Agent token is long-lived (crypto/rand 64 hex). JWT is for user-facing API, expires in 30 days. A background daemon should not require re-authentication. |
| UNIQUE(user_id, machine_id) | One agent per machine per user. Setup is idempotent — re-running returns the existing token. |
| No cobra | Simple `os.Args[1]` switch for setup/run — minimal and consistent with the project style. |

---

## Verification (End-to-End)

1. Run migrations → tables created
2. `POST /api/auth/signup` → get JWT
3. `POST /api/agents/register` with JWT → get agent token
4. Start agent with token → WebSocket connection succeeds
5. Full flow: `mydeploy setup --server ...` → `mydeploy run` → deploy via API
