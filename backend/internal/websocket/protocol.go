package websocket

import "github.com/google/uuid"

// MessageType represents the WebSocket message type
type MessageType string

const (
	TypeRequest  MessageType = "request"
	TypeResponse MessageType = "response"
	TypeEvent    MessageType = "event"
)

// Message is the standard WebSocket message format between panel and agents
type Message struct {
	ID      string      `json:"id"`
	Type    MessageType `json:"type"`
	Action  string      `json:"action"`
	Payload any         `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewRequest creates a new request message
func NewRequest(action string, payload any) *Message {
	return &Message{
		ID:      uuid.New().String(),
		Type:    TypeRequest,
		Action:  action,
		Payload: payload,
	}
}

// NewResponse creates a response message for a given request ID
func NewResponse(requestID string, payload any, err string) *Message {
	return &Message{
		ID:      requestID,
		Type:    TypeResponse,
		Action:  "",
		Payload: payload,
		Error:   err,
	}
}

// NewEvent creates an event message (unsolicited from agent)
func NewEvent(action string, payload any) *Message {
	return &Message{
		ID:      uuid.New().String(),
		Type:    TypeEvent,
		Action:  action,
		Payload: payload,
	}
}

// Action constants
const (
	// Agent lifecycle
	ActionAgentHeartbeat = "agent.heartbeat"

	// IPMI
	ActionIPMIPowerStatus = "ipmi.power.status"
	ActionIPMIPowerOn     = "ipmi.power.on"
	ActionIPMIPowerOff    = "ipmi.power.off"
	ActionIPMIPowerReset  = "ipmi.power.reset"
	ActionIPMIPowerCycle  = "ipmi.power.cycle"
	ActionIPMISensors     = "ipmi.sensors"
	ActionIPMIKVMStart    = "ipmi.kvm.start"
	ActionIPMIKVMStop     = "ipmi.kvm.stop"
	ActionIPMIUserCreate  = "ipmi.user.create"
	ActionIPMIUserDelete  = "ipmi.user.delete"

	// PXE
	ActionPXEPrepare = "pxe.prepare"
	ActionPXEStatus  = "pxe.status"
	ActionPXECleanup = "pxe.cleanup"

	// RAID
	ActionRAIDConfigure = "raid.configure"
	ActionRAIDStatus    = "raid.status"

	// Inventory
	ActionInventoryScan   = "inventory.scan"
	ActionInventoryPXE    = "inventory.pxe"
	ActionInventoryResult = "inventory.result"

	// Switch
	ActionSwitchProvision  = "switch.provision"
	ActionSwitchStatus     = "switch.status"
	ActionSwitchDHCPRelay  = "switch.dhcp_relay"
	ActionSwitchTest       = "switch.test"
	ActionSwitchPortAdmin      = "switch.port_admin"
	ActionSwitchVLANProvision  = "switch.vlan_provision"

	// SNMP
	ActionSNMPPoll = "snmp.poll"
	ActionSNMPData = "snmp.data"

	// Discovery
	ActionDiscoveryStart  = "discovery.start"
	ActionDiscoveryStop   = "discovery.stop"
	ActionDiscoveryResult = "discovery.result"
	ActionDiscoveryStatus = "discovery.status"
)

// HeartbeatPayload is sent by the agent periodically
type HeartbeatPayload struct {
	Version    string   `json:"version"`
	Uptime     int64    `json:"uptime"` // seconds
	Hostname   string   `json:"hostname"`
	LoadAvg    float64  `json:"load_avg"`
	Capabilities []string `json:"capabilities"`
}

// PowerPayload is sent for power control commands
type PowerPayload struct {
	IPMIIP   string `json:"ipmi_ip"`
	IPMIUser string `json:"ipmi_user"`
	IPMIPass string `json:"ipmi_pass"`
	BMCType  string `json:"bmc_type"`
}

// PowerStatusPayload is returned for power status queries
type PowerStatusPayload struct {
	Status string `json:"status"` // on/off/unknown
}

// SensorPayload is returned for IPMI sensor data
type SensorPayload struct {
	Sensors []SensorReading `json:"sensors"`
}

type SensorReading struct {
	Name  string  `json:"name"`
	Type  string  `json:"type"` // temperature/fan/voltage/power
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// KVMStartPayload initiates a KVM session
type KVMStartPayload struct {
	IPMIIP   string `json:"ipmi_ip"`
	IPMIUser string `json:"ipmi_user"`
	IPMIPass string `json:"ipmi_pass"`
	BMCType  string `json:"bmc_type"`
}

// KVMStartResponse returns the WebSocket path for the KVM session
type KVMStartResponse struct {
	SessionID string `json:"session_id"`
	Port      int    `json:"port"` // local port on agent
}

// PXEPreparePayload instructs the agent to set up PXE boot for a server
type PXEPreparePayload struct {
	ServerMAC    string `json:"server_mac"`
	ServerIP     string `json:"server_ip"`
	KernelURL    string `json:"kernel_url"`
	InitrdURL    string `json:"initrd_url"`
	BootArgs     string `json:"boot_args"`
	PreseedURL   string `json:"preseed_url"`
}

// PXEStatusPayload reports installation progress
type PXEStatusPayload struct {
	ServerID string `json:"server_id"`
	Status   string `json:"status"`
	Progress int    `json:"progress"` // 0-100
	Message  string `json:"message"`
}
