CREATE TABLE IF NOT EXISTS agent_bootstrap_tokens (
    token       TEXT PRIMARY KEY,
    user_id     UUID NOT NULL,
    agent_name  TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_bootstrap_tokens_user_id
    ON agent_bootstrap_tokens(user_id);

CREATE INDEX IF NOT EXISTS idx_agent_bootstrap_tokens_expires_at
    ON agent_bootstrap_tokens(expires_at);
