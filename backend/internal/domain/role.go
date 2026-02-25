package domain

import (
	"time"

	"github.com/google/uuid"
)

// Permission constants
const (
	PermServerView        = "server.view"
	PermServerCreate      = "server.create"
	PermServerEdit        = "server.edit"
	PermServerDelete      = "server.delete"
	PermServerPower       = "server.power"
	PermIPMIKVM           = "ipmi.kvm"
	PermIPMISensors       = "ipmi.sensors"
	PermOSReinstall       = "os.reinstall"
	PermOSProfileManage   = "os.profile.manage"
	PermRAIDManage        = "raid.manage"
	PermBandwidthView     = "bandwidth.view"
	PermSwitchManage      = "switch.manage"
	PermInventoryView     = "inventory.view"
	PermInventoryScan     = "inventory.scan"
	PermUserView          = "user.view"
	PermUserManage        = "user.manage"
	PermRoleManage        = "role.manage"
	PermTenantView        = "tenant.view"
	PermTenantManage      = "tenant.manage"
	PermAgentManage       = "agent.manage"
	PermIPManage          = "ip.manage"
	PermAuditView         = "audit.view"
	PermSettingsManage    = "settings.manage"
	PermScriptManage      = "script.manage"
	PermDiskLayoutManage  = "disk_layout.manage"
	PermDiscoveryView     = "discovery.view"
	PermDiscoveryManage   = "discovery.manage"
)

// AllPermissions lists every available permission
var AllPermissions = []string{
	PermServerView, PermServerCreate, PermServerEdit, PermServerDelete, PermServerPower,
	PermIPMIKVM, PermIPMISensors,
	PermOSReinstall, PermOSProfileManage,
	PermRAIDManage,
	PermBandwidthView, PermSwitchManage,
	PermInventoryView, PermInventoryScan,
	PermUserView, PermUserManage, PermRoleManage,
	PermTenantView, PermTenantManage,
	PermAgentManage,
	PermIPManage,
	PermAuditView,
	PermSettingsManage,
	PermScriptManage, PermDiskLayoutManage,
	PermDiscoveryView, PermDiscoveryManage,
}

type Role struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	Name        string     `json:"name" db:"name"`
	Permissions []string   `json:"permissions" db:"permissions"`
	IsSystem    bool       `json:"is_system" db:"is_system"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// HasPermission checks if the role has a specific permission
func (r *Role) HasPermission(perm string) bool {
	for _, p := range r.Permissions {
		if p == perm || p == "*" {
			return true
		}
	}
	return false
}
