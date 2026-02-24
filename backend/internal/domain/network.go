package domain

import (
	"time"

	"github.com/google/uuid"
)

type Switch struct {
	ID            uuid.UUID `json:"id" db:"id"`
	AgentID       uuid.UUID `json:"agent_id" db:"agent_id"`
	Name          string    `json:"name" db:"name"`
	IP            string    `json:"ip" db:"ip"`
	SNMPCommunity string   `json:"snmp_community" db:"snmp_community"`
	SNMPVersion   string    `json:"snmp_version" db:"snmp_version"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type SwitchPort struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	SwitchID    uuid.UUID  `json:"switch_id" db:"switch_id"`
	ServerID    *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	PortIndex   int        `json:"port_index" db:"port_index"`
	PortName    string     `json:"port_name" db:"port_name"`
	SpeedMbps   int        `json:"speed_mbps" db:"speed_mbps"`
	Description string     `json:"description" db:"description"`
}

type IPPool struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	Network     string     `json:"network" db:"network"`
	Gateway     string     `json:"gateway" db:"gateway"`
	Description string     `json:"description" db:"description"`
}

type IPAddress struct {
	ID       uuid.UUID  `json:"id" db:"id"`
	PoolID   uuid.UUID  `json:"pool_id" db:"pool_id"`
	Address  string     `json:"address" db:"address"`
	ServerID *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	Status   string     `json:"status" db:"status"` // available/assigned/reserved
	Note     string     `json:"note" db:"note"`
}
