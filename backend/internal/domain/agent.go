package domain

import (
	"time"

	"github.com/google/uuid"
)

type AgentStatus string

const (
	AgentStatusOnline  AgentStatus = "online"
	AgentStatusOffline AgentStatus = "offline"
	AgentStatusError   AgentStatus = "error"
)

type Agent struct {
	ID           uuid.UUID   `json:"id" db:"id"`
	Name         string      `json:"name" db:"name"`
	Location     string      `json:"location" db:"location"`
	TokenHash    string      `json:"-" db:"token_hash"`
	Status       AgentStatus `json:"status" db:"status"`
	LastSeen     *time.Time  `json:"last_seen,omitempty" db:"last_seen"`
	Version      string      `json:"version" db:"version"`
	Capabilities []string    `json:"capabilities" db:"capabilities"`
	CreatedAt    time.Time   `json:"created_at" db:"created_at"`
}

type AgentCreateRequest struct {
	Name         string   `json:"name" binding:"required"`
	Location     string   `json:"location"`
	Capabilities []string `json:"capabilities"`
}

type AgentCreateResponse struct {
	Agent *Agent `json:"agent"`
	Token string `json:"token"` // returned only on creation
}
