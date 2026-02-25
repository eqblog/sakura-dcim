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
	Vendor        string    `json:"vendor" db:"vendor"`
	Model         string    `json:"model" db:"model"`
	SNMPCommunity string    `json:"snmp_community" db:"snmp_community"`
	SNMPVersion   string    `json:"snmp_version" db:"snmp_version"`
	SSHUser       string    `json:"ssh_user" db:"ssh_user"`
	SSHPass       string    `json:"ssh_pass,omitempty" db:"ssh_pass"`
	SSHPort       int       `json:"ssh_port" db:"ssh_port"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type SwitchPort struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	SwitchID    uuid.UUID  `json:"switch_id" db:"switch_id"`
	ServerID    *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	PortIndex   int        `json:"port_index" db:"port_index"`
	PortName    string     `json:"port_name" db:"port_name"`
	SpeedMbps   int        `json:"speed_mbps" db:"speed_mbps"`
	VlanID       int        `json:"vlan_id" db:"vlan_id"`
	PortMode     string     `json:"port_mode" db:"port_mode"`
	NativeVlanID int        `json:"native_vlan_id" db:"native_vlan_id"`
	TrunkVlans   string     `json:"trunk_vlans" db:"trunk_vlans"`
	AdminStatus  string     `json:"admin_status" db:"admin_status"`
	OperStatus  string     `json:"oper_status" db:"oper_status"`
	Description string     `json:"description" db:"description"`
	LastPolled  *time.Time `json:"last_polled,omitempty" db:"last_polled"`
}

// SwitchPortWithSwitch extends SwitchPort with parent switch info for display purposes.
type SwitchPortWithSwitch struct {
	SwitchPort
	SwitchName string `json:"switch_name"`
	SwitchIP   string `json:"switch_ip"`
}

type IPPool struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	Network     string     `json:"network" db:"network"`
	Gateway     string     `json:"gateway" db:"gateway"`
	Netmask     string     `json:"netmask" db:"netmask"`
	VRF         string     `json:"vrf" db:"vrf"`
	Nameservers []string   `json:"nameservers" db:"nameservers"`
	Description      string     `json:"description" db:"description"`
	Priority         int        `json:"priority" db:"priority"`
	RDNSServer       string     `json:"rdns_server" db:"rdns_server"`
	Notes            string     `json:"notes" db:"notes"`
	SwitchAutomation bool       `json:"switch_automation" db:"switch_automation"`
	VlanID           int        `json:"vlan_id" db:"vlan_id"`
	VlanRangeStart   int        `json:"vlan_range_start" db:"vlan_range_start"`
	VlanRangeEnd     int        `json:"vlan_range_end" db:"vlan_range_end"`
	VlanMode         string     `json:"vlan_mode" db:"vlan_mode"`               // "access", "trunk_native", "trunk"
	NativeVlanID     int        `json:"native_vlan_id" db:"native_vlan_id"`
	TrunkVlans       string     `json:"trunk_vlans" db:"trunk_vlans"`
	VlanAllocation   string     `json:"vlan_allocation" db:"vlan_allocation"` // "fixed" or "auto_range"
	ParentID         *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	PoolType         string     `json:"pool_type" db:"pool_type"` // "ip_pool" or "subnet"
	TotalIPs         int        `json:"total_ips" db:"-"`
	UsedIPs          int        `json:"used_ips" db:"-"`
	ChildCount       int        `json:"child_count" db:"-"`
}

type IPAddress struct {
	ID       uuid.UUID  `json:"id" db:"id"`
	PoolID   uuid.UUID  `json:"pool_id" db:"pool_id"`
	Address  string     `json:"address" db:"address"`
	ServerID *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	Status   string     `json:"status" db:"status"` // available/assigned/reserved
	Note     string     `json:"note" db:"note"`
}

type IPPoolCreateRequest struct {
	TenantID       *uuid.UUID `json:"tenant_id"`
	ParentID       *uuid.UUID `json:"parent_id"`
	Network        string     `json:"network" binding:"required"`
	Gateway        string     `json:"gateway" binding:"required"`
	Description    string     `json:"description"`
	PoolType       string     `json:"pool_type"`
	ReserveGateway bool       `json:"reserve_gateway"`
}

type IPAddressCreateRequest struct {
	Address  string     `json:"address" binding:"required"`
	ServerID *uuid.UUID `json:"server_id"`
	Status   string     `json:"status"`
	Note     string     `json:"note"`
}

type IPAddressUpdateRequest struct {
	ServerID *uuid.UUID `json:"server_id"`
	Status   *string    `json:"status"`
	Note     *string    `json:"note"`
}

// DHCPRelayRequest is the API payload for configuring DHCP relay on a switch interface.
type DHCPRelayRequest struct {
	InterfaceName string `json:"interface_name" binding:"required"` // e.g. "Vlan100", "irb.100"
	DHCPServerIP  string `json:"dhcp_server_ip" binding:"required"` // DHCP server to relay to
	RelayGroup    string `json:"relay_group"`                       // JunOS relay group name
	Remove        bool   `json:"remove"`                            // true = remove relay config
}

// PortAdminRequest is the API payload for toggling port admin status.
type PortAdminRequest struct {
	Status string `json:"status" binding:"required,oneof=up down"`
}

// VLANSummary represents an aggregated VLAN entry across switch ports.
type VLANSummary struct {
	VlanID    int      `json:"vlan_id"`
	PortCount int      `json:"port_count"`
	Ports     []string `json:"ports"`
}

// PortTrafficSummary holds traffic totals for a single port.
type PortTrafficSummary struct {
	TrafficTodayIn  uint64 `json:"traffic_today_in"`
	TrafficTodayOut uint64 `json:"traffic_today_out"`
	TrafficMonthIn  uint64 `json:"traffic_month_in"`
	TrafficMonthOut uint64 `json:"traffic_month_out"`
}

// SwitchBandwidthMap maps port_id to its traffic summary.
type SwitchBandwidthMap map[string]PortTrafficSummary

// SensorDataPoint represents a time-series sensor reading stored in InfluxDB
type SensorDataPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	SensorName string    `json:"sensor_name"`
	SensorType string    `json:"sensor_type"`
	Value      float64   `json:"value"`
	Status     string    `json:"status"`
}

// BandwidthDataPoint represents a single bandwidth measurement
type BandwidthDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	InBytes   uint64    `json:"in_bytes"`
	OutBytes  uint64    `json:"out_bytes"`
	InBps     float64   `json:"in_bps"`
	OutBps    float64   `json:"out_bps"`
}

// BandwidthSummary contains aggregated bandwidth stats
type BandwidthSummary struct {
	PortID      uuid.UUID            `json:"port_id"`
	PortName    string               `json:"port_name"`
	ServerID    *uuid.UUID           `json:"server_id,omitempty"`
	SpeedMbps   int                  `json:"speed_mbps"`
	In95th      float64              `json:"in_95th_bps"`
	Out95th     float64              `json:"out_95th_bps"`
	InAvg       float64              `json:"in_avg_bps"`
	OutAvg      float64              `json:"out_avg_bps"`
	InMax       float64              `json:"in_max_bps"`
	OutMax      float64              `json:"out_max_bps"`
	DataPoints  []BandwidthDataPoint `json:"data_points,omitempty"`
}
