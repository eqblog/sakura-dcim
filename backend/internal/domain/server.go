package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type ServerStatus string

const (
	ServerStatusActive       ServerStatus = "active"
	ServerStatusProvisioning ServerStatus = "provisioning"
	ServerStatusReinstalling ServerStatus = "reinstalling"
	ServerStatusOffline      ServerStatus = "offline"
	ServerStatusError        ServerStatus = "error"
)

type BMCType string

const (
	BMCGeneric    BMCType = "generic"
	BMCDellIDRAC  BMCType = "dell_idrac"
	BMCHPiLO      BMCType = "hp_ilo"
	BMCSupermicro BMCType = "supermicro"
	BMCLenovoXCC  BMCType = "lenovo_xcc"
	BMCHuaweiIBMC BMCType = "huawei_ibmc"
)

// AllBMCTypes lists valid BMC types for validation
var AllBMCTypes = []BMCType{
	BMCGeneric, BMCDellIDRAC, BMCHPiLO, BMCSupermicro, BMCLenovoXCC, BMCHuaweiIBMC,
}

// DetectBMCType infers BMC type from a system vendor string (e.g. from discovery)
func DetectBMCType(vendor string) BMCType {
	v := strings.ToLower(vendor)
	switch {
	case strings.Contains(v, "dell"):
		return BMCDellIDRAC
	case strings.Contains(v, "hp"), strings.Contains(v, "hpe"), strings.Contains(v, "hewlett"):
		return BMCHPiLO
	case strings.Contains(v, "supermicro"):
		return BMCSupermicro
	case strings.Contains(v, "lenovo"):
		return BMCLenovoXCC
	case strings.Contains(v, "huawei"):
		return BMCHuaweiIBMC
	default:
		return BMCGeneric
	}
}

type Server struct {
	ID       uuid.UUID    `json:"id" db:"id"`
	TenantID *uuid.UUID   `json:"tenant_id,omitempty" db:"tenant_id"`
	AgentID  *uuid.UUID   `json:"agent_id,omitempty" db:"agent_id"`
	Hostname string       `json:"hostname" db:"hostname"`
	Label    string       `json:"label" db:"label"`
	Status   ServerStatus `json:"status" db:"status"`
	// Network
	PrimaryIP  string  `json:"primary_ip" db:"primary_ip"`
	IPMIIP     string  `json:"ipmi_ip" db:"ipmi_ip"`
	IPMIUser   string  `json:"ipmi_user,omitempty" db:"ipmi_user"` // encrypted
	IPMIPass   string  `json:"ipmi_pass,omitempty" db:"ipmi_pass"` // encrypted
	MACAddress string  `json:"mac_address" db:"mac_address"`
	BMCType    BMCType `json:"bmc_type" db:"bmc_type"`
	// Hardware summary
	CPUModel string `json:"cpu_model" db:"cpu_model"`
	CPUCores int    `json:"cpu_cores" db:"cpu_cores"`
	RAMMB    int64  `json:"ram_mb" db:"ram_mb"`
	// Tags
	Tags []string `json:"tags" db:"tags"`
	// Metadata
	Notes     string    `json:"notes" db:"notes"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Joined
	Agent *Agent `json:"agent,omitempty" db:"-"`
}

type ServerCreateRequest struct {
	AgentID    *uuid.UUID `json:"agent_id"`
	Hostname   string     `json:"hostname" binding:"required"`
	Label      string     `json:"label"`
	PrimaryIP  string     `json:"primary_ip"`
	IPMIIP     string     `json:"ipmi_ip"`
	IPMIUser   string     `json:"ipmi_user"`
	IPMIPass   string     `json:"ipmi_pass"`
	MACAddress string     `json:"mac_address"`
	BMCType    BMCType    `json:"bmc_type"`
	Tags       []string   `json:"tags"`
	Notes      string     `json:"notes"`
}

type ServerUpdateRequest struct {
	Hostname   *string    `json:"hostname"`
	Label      *string    `json:"label"`
	AgentID    *uuid.UUID `json:"agent_id"`
	PrimaryIP  *string    `json:"primary_ip"`
	IPMIIP     *string    `json:"ipmi_ip"`
	IPMIUser   *string    `json:"ipmi_user"`
	IPMIPass   *string    `json:"ipmi_pass"`
	MACAddress *string    `json:"mac_address"`
	BMCType    *BMCType   `json:"bmc_type"`
	Tags       *[]string  `json:"tags"`
	Notes      *string    `json:"notes"`
}

type ServerListParams struct {
	TenantID *uuid.UUID
	AgentID  *uuid.UUID
	Status   *ServerStatus
	Tags     []string
	Search   string
	Page     int
	PageSize int
}

type ServerDisk struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ServerID  uuid.UUID `json:"server_id" db:"server_id"`
	Slot      string    `json:"slot" db:"slot"`
	Model     string    `json:"model" db:"model"`
	Serial    string    `json:"serial" db:"serial"`
	SizeBytes int64     `json:"size_bytes" db:"size_bytes"`
	Type      string    `json:"type" db:"type"` // ssd/hdd/nvme
	Health    string    `json:"health" db:"health"`
}

// ServerInventory represents a collected hardware component entry
type ServerInventory struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ServerID    uuid.UUID `json:"server_id" db:"server_id"`
	Component   string    `json:"component" db:"component"` // cpu, memory, disks, network, system
	Details     any       `json:"details" db:"details"`     // JSONB
	CollectedAt time.Time `json:"collected_at" db:"collected_at"`
}

// InventoryResult is the full inventory response grouped by component
type InventoryResult struct {
	ServerID    uuid.UUID          `json:"server_id"`
	Components  []ServerInventory  `json:"components"`
	CollectedAt *time.Time         `json:"collected_at,omitempty"`
}
