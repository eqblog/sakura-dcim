package domain

import (
	"time"

	"github.com/google/uuid"
)

type DiscoverySessionStatus string

const (
	DiscoveryStatusActive  DiscoverySessionStatus = "active"
	DiscoveryStatusStopped DiscoverySessionStatus = "stopped"
)

type DiscoverySession struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	AgentID       uuid.UUID              `json:"agent_id" db:"agent_id"`
	Status        DiscoverySessionStatus `json:"status" db:"status"`
	CallbackToken string                 `json:"-" db:"callback_token"`
	DHCPRange     string                 `json:"dhcp_range" db:"dhcp_range"`
	StartedBy     *uuid.UUID             `json:"started_by,omitempty" db:"started_by"`
	StartedAt     time.Time              `json:"started_at" db:"started_at"`
	StoppedAt     *time.Time             `json:"stopped_at,omitempty" db:"stopped_at"`
}

type DiscoveredServerStatus string

const (
	DiscoveredStatusPending  DiscoveredServerStatus = "pending"
	DiscoveredStatusApproved DiscoveredServerStatus = "approved"
	DiscoveredStatusRejected DiscoveredServerStatus = "rejected"
)

type DiscoveredServer struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	SessionID     uuid.UUID              `json:"session_id" db:"session_id"`
	AgentID       uuid.UUID              `json:"agent_id" db:"agent_id"`
	MACAddress    string                 `json:"mac_address" db:"mac_address"`
	IPAddress     string                 `json:"ip_address" db:"ip_address"`
	Status        DiscoveredServerStatus `json:"status" db:"status"`
	SystemVendor  string                 `json:"system_vendor" db:"system_vendor"`
	SystemProduct string                 `json:"system_product" db:"system_product"`
	SystemSerial  string                 `json:"system_serial" db:"system_serial"`
	CPUModel      string                 `json:"cpu_model" db:"cpu_model"`
	CPUCores      int                    `json:"cpu_cores" db:"cpu_cores"`
	CPUSockets    int                    `json:"cpu_sockets" db:"cpu_sockets"`
	RAMMB         int64                  `json:"ram_mb" db:"ram_mb"`
	DiskCount     int                    `json:"disk_count" db:"disk_count"`
	DiskTotalGB   int64                  `json:"disk_total_gb" db:"disk_total_gb"`
	NICCount      int                    `json:"nic_count" db:"nic_count"`
	RawInventory  any                    `json:"raw_inventory" db:"raw_inventory"`
	BMCIP         string                 `json:"bmc_ip" db:"bmc_ip"`
	ApprovedBy    *uuid.UUID             `json:"approved_by,omitempty" db:"approved_by"`
	ServerID      *uuid.UUID             `json:"server_id,omitempty" db:"server_id"`
	DiscoveredAt  time.Time              `json:"discovered_at" db:"discovered_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

type DiscoveryStartRequest struct {
	DHCPRangeStart string `json:"dhcp_range_start" binding:"required"`
	DHCPRangeEnd   string `json:"dhcp_range_end" binding:"required"`
	Gateway        string `json:"gateway" binding:"required"`
	Netmask        string `json:"netmask" binding:"required"`
}

type DiscoveryApproveRequest struct {
	Hostname string     `json:"hostname" binding:"required"`
	Label    string     `json:"label"`
	AgentID  *uuid.UUID `json:"agent_id"`
	IPMIIP   string     `json:"ipmi_ip"`
	IPMIUser string     `json:"ipmi_user"`
	IPMIPass string     `json:"ipmi_pass"`
	Tags     []string   `json:"tags"`
	Notes    string     `json:"notes"`
}

type DiscoveredServerListParams struct {
	AgentID  *uuid.UUID
	Status   *DiscoveredServerStatus
	Search   string
	Page     int
	PageSize int
}
