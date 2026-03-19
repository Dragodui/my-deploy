CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT UNIQUE NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    password    TEXT NOT NULL,  
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);