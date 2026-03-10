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