CREATE TABLE agents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    token       TEXT UNIQUE NOT NULL,  
    name        TEXT NOT NULL,        
    machine_id  TEXT NOT NULL,       
    last_seen   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, machine_id)
);
CREATE INDEX idx_agents_token ON agents(token);