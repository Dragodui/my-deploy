package models

import "time"

type Agent struct {
	//   id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	// user_id     UUID NOT NULL REFERENCES users(id),
	// token       TEXT UNIQUE NOT NULL,  -- crypto/rand 32-byte hex
	// name        TEXT NOT NULL,         -- hostname
	// machine_id  TEXT NOT NULL,         -- machine fingerprint
	// last_seen   TIMESTAMPTZ,
	// created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
	// UNIQUE(user_id, machine_id)

	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	Name      string    `json:"name" db:"name"`
	MachineID string    `json:"machine_id" db:"machine_id"`
	LastSeen  time.Time `json:"last_seen" db:"last_seen"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type AgentBootstrapToken struct {
	Token     string     `json:"token" db:"token"`
	UserID    string     `json:"user_id" db:"user_id"`
	AgentName string     `json:"agent_name" db:"agent_name"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}
