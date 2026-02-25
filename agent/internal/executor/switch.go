package executor

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// SwitchExecutor handles switch port provisioning and status queries via SSH.
type SwitchExecutor struct {
	logger *zap.Logger
}

func NewSwitchExecutor(logger *zap.Logger) *SwitchExecutor {
	return &SwitchExecutor{logger: logger}
}

type SwitchProvisionPayload struct {
	SwitchIP     string `json:"switch_ip"`
	SSHUser      string `json:"ssh_user"`
	SSHPass      string `json:"ssh_pass"`
	SSHPort      int    `json:"ssh_port"`
	Vendor       string `json:"vendor"`
	PortName     string `json:"port_name"`
	VlanID       int    `json:"vlan_id"`
	PortMode     string `json:"port_mode"`
	NativeVlanID int    `json:"native_vlan_id"`
	TrunkVlans   string `json:"trunk_vlans"`
	SpeedMbps    int    `json:"speed_mbps"`
	AdminStatus  string `json:"admin_status"`
	Description  string `json:"description"`
}

// SwitchPortAdminPayload contains parameters for toggling port admin status.
type SwitchPortAdminPayload struct {
	SwitchIP string `json:"switch_ip"`
	SSHUser  string `json:"ssh_user"`
	SSHPass  string `json:"ssh_pass"`
	SSHPort  int    `json:"ssh_port"`
	Vendor   string `json:"vendor"`
	PortName string `json:"port_name"`
	Status   string `json:"status"` // "up" or "down"
}

type SwitchStatusPayload struct {
	SwitchIP      string `json:"switch_ip"`
	SSHUser       string `json:"ssh_user"`
	SSHPass       string `json:"ssh_pass"`
	SSHPort       int    `json:"ssh_port"`
	Vendor        string `json:"vendor"`
	SNMPCommunity string `json:"snmp_community"`
	SNMPVersion   string `json:"snmp_version"`
	PortName      string `json:"port_name"`
	PortIndex     int    `json:"port_index"`
}

// HandleSwitchProvision configures a switch port via SSH.
func (e *SwitchExecutor) HandleSwitchProvision(raw json.RawMessage) (interface{}, error) {
	var p SwitchProvisionPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	e.logger.Info("provisioning switch port",
		zap.String("switch_ip", p.SwitchIP),
		zap.String("port", p.PortName),
		zap.Int("vlan", p.VlanID),
	)

	commands := generateProvisionCommands(p.Vendor, &p)
	if len(commands) == 0 {
		return nil, fmt.Errorf("unsupported vendor: %s", p.Vendor)
	}

	output, err := e.execSSH(p.SwitchIP, p.SSHPort, p.SSHUser, p.SSHPass, commands)
	if err != nil {
		return nil, fmt.Errorf("ssh execution failed: %w", err)
	}

	return map[string]string{
		"status": "provisioned",
		"output": output,
	}, nil
}

// HandleSwitchStatus queries switch port status via SSH.
func (e *SwitchExecutor) HandleSwitchStatus(raw json.RawMessage) (interface{}, error) {
	var p SwitchStatusPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	e.logger.Info("querying switch port status",
		zap.String("switch_ip", p.SwitchIP),
		zap.String("port", p.PortName),
	)

	commands := generateStatusCommands(p.Vendor, p.PortName)
	if len(commands) == 0 {
		return nil, fmt.Errorf("unsupported vendor: %s", p.Vendor)
	}

	output, err := e.execSSH(p.SwitchIP, p.SSHPort, p.SSHUser, p.SSHPass, commands)
	if err != nil {
		return nil, fmt.Errorf("ssh execution failed: %w", err)
	}

	return map[string]string{
		"status": "ok",
		"output": output,
	}, nil
}

// HandleSwitchPortAdmin toggles a switch port admin status (shutdown/no shutdown) via SSH.
func (e *SwitchExecutor) HandleSwitchPortAdmin(raw json.RawMessage) (interface{}, error) {
	var p SwitchPortAdminPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	e.logger.Info("toggling port admin status",
		zap.String("switch_ip", p.SwitchIP),
		zap.String("port", p.PortName),
		zap.String("status", p.Status),
	)

	commands := generatePortAdminCommands(p.Vendor, p.PortName, p.Status)
	if len(commands) == 0 {
		return nil, fmt.Errorf("unsupported vendor: %s", p.Vendor)
	}

	output, err := e.execSSH(p.SwitchIP, p.SSHPort, p.SSHUser, p.SSHPass, commands)
	if err != nil {
		return nil, fmt.Errorf("ssh execution failed: %w", err)
	}

	return map[string]string{
		"status": p.Status,
		"output": output,
	}, nil
}

// generatePortAdminCommands creates vendor-specific CLI commands for port admin toggle.
func generatePortAdminCommands(vendor, portName, status string) []string {
	shutdown := status == "down"
	switch normalizeVendor(vendor) {
	case "cisco_ios":
		cmds := []string{"configure terminal", fmt.Sprintf("interface %s", portName)}
		if shutdown {
			cmds = append(cmds, "shutdown")
		} else {
			cmds = append(cmds, "no shutdown")
		}
		return append(cmds, "end", "write memory")
	case "cisco_nxos":
		cmds := []string{"configure terminal", fmt.Sprintf("interface %s", portName)}
		if shutdown {
			cmds = append(cmds, "shutdown")
		} else {
			cmds = append(cmds, "no shutdown")
		}
		return append(cmds, "end", "copy running-config startup-config")
	case "junos":
		if shutdown {
			return []string{fmt.Sprintf("set interfaces %s disable", portName), "commit and-quit"}
		}
		return []string{fmt.Sprintf("delete interfaces %s disable", portName), "commit and-quit"}
	case "arista_eos":
		cmds := []string{"configure", fmt.Sprintf("interface %s", portName)}
		if shutdown {
			cmds = append(cmds, "shutdown")
		} else {
			cmds = append(cmds, "no shutdown")
		}
		return append(cmds, "end", "write memory")
	case "sonic":
		if shutdown {
			return []string{fmt.Sprintf("sudo config interface shutdown %s", portName), "sudo config save -y"}
		}
		return []string{fmt.Sprintf("sudo config interface startup %s", portName), "sudo config save -y"}
	case "cumulus":
		if shutdown {
			return []string{fmt.Sprintf("net add interface %s link down", portName), "net commit"}
		}
		return []string{fmt.Sprintf("net del interface %s link down", portName), "net commit"}
	default:
		return nil
	}
}

func (e *SwitchExecutor) execSSH(host string, port int, user, pass string, commands []string) (string, error) {
	if port == 0 {
		port = 22
	}
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("ssh session: %w", err)
	}
	defer session.Close()

	cmdStr := strings.Join(commands, "\n")
	out, err := session.CombinedOutput(cmdStr)
	if err != nil {
		return string(out), fmt.Errorf("ssh exec: %s: %w", string(out), err)
	}
	return string(out), nil
}

// generateCiscoPortModeCmds generates switchport mode commands for Cisco IOS/NX-OS/Arista EOS.
func generateCiscoPortModeCmds(p *SwitchProvisionPayload) []string {
	var cmds []string
	switch p.PortMode {
	case "trunk":
		cmds = append(cmds, "switchport mode trunk")
		if p.TrunkVlans != "" {
			cmds = append(cmds, fmt.Sprintf("switchport trunk allowed vlan %s", p.TrunkVlans))
		}
	case "trunk_native":
		cmds = append(cmds, "switchport mode trunk")
		if p.NativeVlanID > 0 {
			cmds = append(cmds, fmt.Sprintf("switchport trunk native vlan %d", p.NativeVlanID))
		}
		if p.TrunkVlans != "" {
			cmds = append(cmds, fmt.Sprintf("switchport trunk allowed vlan %s", p.TrunkVlans))
		}
	default: // access
		if p.VlanID > 0 {
			cmds = append(cmds, "switchport mode access", fmt.Sprintf("switchport access vlan %d", p.VlanID))
		}
	}
	return cmds
}

// generateProvisionCommands creates vendor-specific CLI commands for port configuration.
func generateProvisionCommands(vendor string, p *SwitchProvisionPayload) []string {
	switch normalizeVendor(vendor) {
	case "cisco_ios":
		cmds := []string{
			"configure terminal",
			fmt.Sprintf("interface %s", p.PortName),
		}
		if p.Description != "" {
			cmds = append(cmds, fmt.Sprintf("description %s", p.Description))
		}
		cmds = append(cmds, generateCiscoPortModeCmds(p)...)
		if p.AdminStatus == "down" {
			cmds = append(cmds, "shutdown")
		} else {
			cmds = append(cmds, "no shutdown")
		}
		cmds = append(cmds, "end", "write memory")
		return cmds

	case "cisco_nxos":
		cmds := []string{
			"configure terminal",
			fmt.Sprintf("interface %s", p.PortName),
		}
		if p.Description != "" {
			cmds = append(cmds, fmt.Sprintf("description %s", p.Description))
		}
		cmds = append(cmds, "switchport")
		cmds = append(cmds, generateCiscoPortModeCmds(p)...)
		if p.SpeedMbps > 0 {
			cmds = append(cmds, fmt.Sprintf("speed %d", p.SpeedMbps))
		}
		if p.AdminStatus == "down" {
			cmds = append(cmds, "shutdown")
		} else {
			cmds = append(cmds, "no shutdown")
		}
		cmds = append(cmds, "end", "copy running-config startup-config")
		return cmds

	case "junos":
		cmds := []string{}
		if p.Description != "" {
			cmds = append(cmds, fmt.Sprintf("set interfaces %s description \"%s\"", p.PortName, p.Description))
		}
		switch p.PortMode {
		case "trunk":
			cmds = append(cmds, fmt.Sprintf("set interfaces %s unit 0 family ethernet-switching interface-mode trunk", p.PortName))
			if p.TrunkVlans != "" {
				for _, v := range strings.Split(p.TrunkVlans, ",") {
					v = strings.TrimSpace(v)
					if v != "" {
						cmds = append(cmds, fmt.Sprintf("set interfaces %s unit 0 family ethernet-switching vlan members vlan%s", p.PortName, v))
					}
				}
			}
		case "trunk_native":
			cmds = append(cmds, fmt.Sprintf("set interfaces %s unit 0 family ethernet-switching interface-mode trunk", p.PortName))
			if p.NativeVlanID > 0 {
				cmds = append(cmds, fmt.Sprintf("set interfaces %s native-vlan-id %d", p.PortName, p.NativeVlanID))
			}
			if p.TrunkVlans != "" {
				for _, v := range strings.Split(p.TrunkVlans, ",") {
					v = strings.TrimSpace(v)
					if v != "" {
						cmds = append(cmds, fmt.Sprintf("set interfaces %s unit 0 family ethernet-switching vlan members vlan%s", p.PortName, v))
					}
				}
			}
		default: // access
			if p.VlanID > 0 {
				cmds = append(cmds, fmt.Sprintf("set interfaces %s unit 0 family ethernet-switching vlan members vlan%d", p.PortName, p.VlanID))
			}
		}
		if p.AdminStatus == "down" {
			cmds = append(cmds, fmt.Sprintf("set interfaces %s disable", p.PortName))
		} else {
			cmds = append(cmds, fmt.Sprintf("delete interfaces %s disable", p.PortName))
		}
		cmds = append(cmds, "commit and-quit")
		return cmds

	case "arista_eos":
		cmds := []string{
			"configure",
			fmt.Sprintf("interface %s", p.PortName),
		}
		if p.Description != "" {
			cmds = append(cmds, fmt.Sprintf("description %s", p.Description))
		}
		cmds = append(cmds, generateCiscoPortModeCmds(p)...)
		if p.AdminStatus == "down" {
			cmds = append(cmds, "shutdown")
		} else {
			cmds = append(cmds, "no shutdown")
		}
		cmds = append(cmds, "end", "write memory")
		return cmds

	case "sonic":
		cmds := []string{}
		switch p.PortMode {
		case "trunk":
			if p.TrunkVlans != "" {
				for _, v := range strings.Split(p.TrunkVlans, ",") {
					v = strings.TrimSpace(v)
					if v != "" {
						cmds = append(cmds, fmt.Sprintf("sudo config vlan member add %s %s --untagged", v, p.PortName))
					}
				}
			}
		default: // access
			if p.VlanID > 0 {
				cmds = append(cmds, fmt.Sprintf("sudo config vlan member add %d %s", p.VlanID, p.PortName))
			}
		}
		if p.AdminStatus == "down" {
			cmds = append(cmds, fmt.Sprintf("sudo config interface shutdown %s", p.PortName))
		} else {
			cmds = append(cmds, fmt.Sprintf("sudo config interface startup %s", p.PortName))
		}
		if p.Description != "" {
			cmds = append(cmds, fmt.Sprintf("sudo config interface description %s \"%s\"", p.PortName, p.Description))
		}
		cmds = append(cmds, "sudo config save -y")
		return cmds

	case "cumulus":
		cmds := []string{}
		cmds = append(cmds, fmt.Sprintf("net add bridge bridge ports %s", p.PortName))
		switch p.PortMode {
		case "trunk":
			if p.TrunkVlans != "" {
				cmds = append(cmds, fmt.Sprintf("net add interface %s bridge trunk vlans %s", p.PortName, p.TrunkVlans))
			}
		case "trunk_native":
			if p.NativeVlanID > 0 {
				cmds = append(cmds, fmt.Sprintf("net add interface %s bridge pvid %d", p.PortName, p.NativeVlanID))
			}
			if p.TrunkVlans != "" {
				cmds = append(cmds, fmt.Sprintf("net add interface %s bridge trunk vlans %s", p.PortName, p.TrunkVlans))
			}
		default: // access
			if p.VlanID > 0 {
				cmds = append(cmds, fmt.Sprintf("net add interface %s bridge access %d", p.PortName, p.VlanID))
			}
		}
		if p.AdminStatus == "down" {
			cmds = append(cmds, fmt.Sprintf("net add interface %s link down", p.PortName))
		} else {
			cmds = append(cmds, fmt.Sprintf("net del interface %s link down", p.PortName))
		}
		if p.Description != "" {
			cmds = append(cmds, fmt.Sprintf("net add interface %s alias \"%s\"", p.PortName, p.Description))
		}
		cmds = append(cmds, "net commit")
		return cmds

	default:
		return nil
	}
}

// generateStatusCommands creates vendor-specific CLI commands for port status queries.
func generateStatusCommands(vendor, portName string) []string {
	switch normalizeVendor(vendor) {
	case "cisco_ios":
		return []string{fmt.Sprintf("show interface %s", portName)}
	case "cisco_nxos":
		return []string{fmt.Sprintf("show interface %s", portName)}
	case "junos":
		return []string{fmt.Sprintf("show interfaces %s", portName)}
	case "arista_eos":
		return []string{fmt.Sprintf("show interfaces %s", portName)}
	case "sonic":
		return []string{fmt.Sprintf("show interfaces status %s", portName)}
	case "cumulus":
		return []string{fmt.Sprintf("net show interface %s", portName)}
	default:
		return nil
	}
}

// SwitchDHCPRelayPayload contains parameters for configuring DHCP relay on a switch interface.
type SwitchDHCPRelayPayload struct {
	SwitchIP      string `json:"switch_ip"`
	SSHUser       string `json:"ssh_user"`
	SSHPass       string `json:"ssh_pass"`
	SSHPort       int    `json:"ssh_port"`
	Vendor        string `json:"vendor"`
	InterfaceName string `json:"interface_name"`
	DHCPServerIP  string `json:"dhcp_server_ip"`
	RelayGroup    string `json:"relay_group"`
	Remove        bool   `json:"remove"`
}

// HandleSwitchDHCPRelay configures or removes DHCP relay on a switch interface via SSH.
func (e *SwitchExecutor) HandleSwitchDHCPRelay(raw json.RawMessage) (interface{}, error) {
	var p SwitchDHCPRelayPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	e.logger.Info("configuring DHCP relay",
		zap.String("switch_ip", p.SwitchIP),
		zap.String("interface", p.InterfaceName),
		zap.String("dhcp_server", p.DHCPServerIP),
		zap.Bool("remove", p.Remove),
	)

	commands := generateDHCPRelayCommands(p.Vendor, &p)
	if len(commands) == 0 {
		return nil, fmt.Errorf("unsupported vendor for DHCP relay: %s", p.Vendor)
	}

	output, err := e.execSSH(p.SwitchIP, p.SSHPort, p.SSHUser, p.SSHPass, commands)
	if err != nil {
		return nil, fmt.Errorf("ssh execution failed: %w", err)
	}

	status := "configured"
	if p.Remove {
		status = "removed"
	}
	return map[string]string{
		"status": status,
		"output": output,
	}, nil
}

// generateDHCPRelayCommands creates vendor-specific CLI commands for DHCP relay configuration.
func generateDHCPRelayCommands(vendor string, p *SwitchDHCPRelayPayload) []string {
	switch normalizeVendor(vendor) {
	case "cisco_ios":
		cmds := []string{
			"configure terminal",
			fmt.Sprintf("interface %s", p.InterfaceName),
		}
		if p.Remove {
			cmds = append(cmds, fmt.Sprintf("no ip helper-address %s", p.DHCPServerIP))
		} else {
			cmds = append(cmds, fmt.Sprintf("ip helper-address %s", p.DHCPServerIP))
		}
		cmds = append(cmds, "end", "write memory")
		return cmds

	case "cisco_nxos":
		cmds := []string{
			"configure terminal",
			fmt.Sprintf("interface %s", p.InterfaceName),
		}
		if p.Remove {
			cmds = append(cmds, fmt.Sprintf("no ip dhcp relay address %s", p.DHCPServerIP))
		} else {
			cmds = append(cmds, fmt.Sprintf("ip dhcp relay address %s", p.DHCPServerIP))
		}
		cmds = append(cmds, "end", "copy running-config startup-config")
		return cmds

	case "junos":
		group := p.RelayGroup
		if group == "" {
			group = "default"
		}
		if p.Remove {
			return []string{
				fmt.Sprintf("delete forwarding-options dhcp-relay group %s interface %s", group, p.InterfaceName),
				"commit and-quit",
			}
		}
		return []string{
			fmt.Sprintf("set forwarding-options dhcp-relay group %s interface %s", group, p.InterfaceName),
			fmt.Sprintf("set forwarding-options dhcp-relay server-group %s %s", group, p.DHCPServerIP),
			"commit and-quit",
		}

	case "arista_eos":
		cmds := []string{
			"configure",
			fmt.Sprintf("interface %s", p.InterfaceName),
		}
		if p.Remove {
			cmds = append(cmds, fmt.Sprintf("no ip helper-address %s", p.DHCPServerIP))
		} else {
			cmds = append(cmds, fmt.Sprintf("ip helper-address %s", p.DHCPServerIP))
		}
		cmds = append(cmds, "end", "write memory")
		return cmds

	case "sonic":
		if p.Remove {
			return []string{
				fmt.Sprintf("sudo config vlan dhcp_relay del %s %s", p.InterfaceName, p.DHCPServerIP),
				"sudo config save -y",
			}
		}
		return []string{
			fmt.Sprintf("sudo config vlan dhcp_relay add %s %s", p.InterfaceName, p.DHCPServerIP),
			"sudo config save -y",
		}

	case "cumulus":
		if p.Remove {
			return []string{
				fmt.Sprintf("net del dhcp relay interface %s", p.InterfaceName),
				fmt.Sprintf("net del dhcp relay server %s", p.DHCPServerIP),
				"net commit",
			}
		}
		return []string{
			fmt.Sprintf("net add dhcp relay interface %s", p.InterfaceName),
			fmt.Sprintf("net add dhcp relay server %s", p.DHCPServerIP),
			"net commit",
		}

	default:
		return nil
	}
}

// SwitchTestPayload contains parameters for testing connectivity to a switch.
type SwitchTestPayload struct {
	SwitchIP      string `json:"switch_ip"`
	SSHUser       string `json:"ssh_user"`
	SSHPass       string `json:"ssh_pass"`
	SSHPort       int    `json:"ssh_port"`
	Vendor        string `json:"vendor"`
	SNMPCommunity string `json:"snmp_community"`
	SNMPVersion   string `json:"snmp_version"`
}

// HandleTestConnection tests SSH and SNMP connectivity to a switch.
func (e *SwitchExecutor) HandleTestConnection(raw json.RawMessage) (interface{}, error) {
	var p SwitchTestPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	e.logger.Info("testing switch connection",
		zap.String("switch_ip", p.SwitchIP),
	)

	result := map[string]interface{}{
		"switch_ip": p.SwitchIP,
	}

	// Test SSH
	sshOK := false
	sshMsg := ""
	commands := generateStatusCommands(p.Vendor, "")
	if len(commands) == 0 {
		commands = []string{"show version"}
	}
	output, err := e.execSSH(p.SwitchIP, p.SSHPort, p.SSHUser, p.SSHPass, commands[:1])
	if err != nil {
		sshMsg = err.Error()
	} else {
		sshOK = true
		sshMsg = "connected"
		if len(output) > 200 {
			output = output[:200] + "..."
		}
		result["ssh_output"] = output
	}
	result["ssh_ok"] = sshOK
	result["ssh_message"] = sshMsg

	// Test SNMP (sysDescr.0)
	snmpOK := false
	snmpMsg := ""
	if p.SNMPCommunity != "" {
		version := "-v2c"
		if p.SNMPVersion == "v3" {
			version = "-v3"
		}
		out, err := exec.Command("snmpget", version, "-c", p.SNMPCommunity, p.SwitchIP, "1.3.6.1.2.1.1.1.0").CombinedOutput()
		if err != nil {
			snmpMsg = fmt.Sprintf("%s: %s", err.Error(), strings.TrimSpace(string(out)))
		} else {
			snmpOK = true
			desc := strings.TrimSpace(string(out))
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			snmpMsg = "connected"
			result["snmp_sysdescr"] = desc
		}
	} else {
		snmpMsg = "no community configured"
	}
	result["snmp_ok"] = snmpOK
	result["snmp_message"] = snmpMsg

	return result, nil
}

// SwitchVLANProvisionPayload contains parameters for VLAN infrastructure provisioning.
type SwitchVLANProvisionPayload struct {
	SwitchIP string `json:"switch_ip"`
	SSHUser  string `json:"ssh_user"`
	SSHPass  string `json:"ssh_pass"`
	SSHPort  int    `json:"ssh_port"`
	Vendor   string `json:"vendor"`
	VlanID   int    `json:"vlan_id"`
	VlanName string `json:"vlan_name"`
	Gateway  string `json:"gateway"` // SVI IP, e.g. "10.0.100.1"
	Netmask  string `json:"netmask"` // e.g. "255.255.255.0"
	VRF      string `json:"vrf"`     // VRF name, empty = no VRF
	DryRun   bool   `json:"dry_run"`
}

type vlanProvisionStep struct {
	Action   string   `json:"action"`
	Commands []string `json:"commands"`
	Status   string   `json:"status"`
	Message  string   `json:"message,omitempty"`
}

// HandleSwitchVLANProvision creates VLAN, SVI/VLANIF, and VRF binding on a switch.
func (e *SwitchExecutor) HandleSwitchVLANProvision(raw json.RawMessage) (interface{}, error) {
	var p SwitchVLANProvisionPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	e.logger.Info("VLAN infrastructure provisioning",
		zap.String("switch_ip", p.SwitchIP),
		zap.Int("vlan_id", p.VlanID),
		zap.String("vrf", p.VRF),
		zap.Bool("dry_run", p.DryRun),
	)

	steps := generateVLANProvisionCommands(p.Vendor, &p)
	if len(steps) == 0 {
		return nil, fmt.Errorf("unsupported vendor for VLAN provisioning: %s", p.Vendor)
	}

	if p.DryRun {
		for i := range steps {
			steps[i].Status = "pending"
		}
		return map[string]interface{}{
			"steps":   steps,
			"dry_run": true,
		}, nil
	}

	// Collect all commands into a single SSH session
	var allCmds []string
	for _, s := range steps {
		allCmds = append(allCmds, s.Commands...)
	}

	output, err := e.execSSH(p.SwitchIP, p.SSHPort, p.SSHUser, p.SSHPass, allCmds)
	if err != nil {
		for i := range steps {
			steps[i].Status = "error"
			steps[i].Message = err.Error()
		}
		return map[string]interface{}{
			"steps":   steps,
			"dry_run": false,
			"output":  output,
		}, fmt.Errorf("ssh execution failed: %w", err)
	}

	for i := range steps {
		steps[i].Status = "ok"
	}
	return map[string]interface{}{
		"steps":   steps,
		"dry_run": false,
		"output":  output,
	}, nil
}

// generateVLANProvisionCommands creates vendor-specific CLI commands for VLAN infrastructure.
func generateVLANProvisionCommands(vendor string, p *SwitchVLANProvisionPayload) []vlanProvisionStep {
	switch normalizeVendor(vendor) {
	case "arista_eos":
		return generateAristaVLANProvision(p)
	case "cisco_nxos":
		return generateNXOSVLANProvision(p)
	case "cisco_ios":
		return generateIOSVLANProvision(p)
	default:
		return nil
	}
}

func generateAristaVLANProvision(p *SwitchVLANProvisionPayload) []vlanProvisionStep {
	var steps []vlanProvisionStep

	// Step 1: Create VLAN
	vlanCmds := []string{"configure", fmt.Sprintf("vlan %d", p.VlanID)}
	if p.VlanName != "" {
		vlanCmds = append(vlanCmds, fmt.Sprintf("name %s", p.VlanName))
	}
	steps = append(steps, vlanProvisionStep{
		Action:   "create_vlan",
		Commands: vlanCmds,
	})

	// Step 2: Create SVI with IP (if gateway provided)
	if p.Gateway != "" && p.Netmask != "" {
		prefix := netmaskToPrefixLen(p.Netmask)
		sviCmds := []string{fmt.Sprintf("interface Vlan%d", p.VlanID)}
		if p.VRF != "" {
			sviCmds = append(sviCmds, fmt.Sprintf("vrf %s", p.VRF))
		}
		sviCmds = append(sviCmds, fmt.Sprintf("ip address %s/%d", p.Gateway, prefix), "no shutdown")
		steps = append(steps, vlanProvisionStep{
			Action:   "create_svi",
			Commands: sviCmds,
		})
	}

	// Save config
	steps = append(steps, vlanProvisionStep{
		Action:   "save_config",
		Commands: []string{"end", "write memory"},
	})

	return steps
}

func generateNXOSVLANProvision(p *SwitchVLANProvisionPayload) []vlanProvisionStep {
	var steps []vlanProvisionStep

	// Step 1: Create VLAN
	vlanCmds := []string{"configure terminal", fmt.Sprintf("vlan %d", p.VlanID)}
	if p.VlanName != "" {
		vlanCmds = append(vlanCmds, fmt.Sprintf("name %s", p.VlanName))
	}
	vlanCmds = append(vlanCmds, "exit")
	steps = append(steps, vlanProvisionStep{
		Action:   "create_vlan",
		Commands: vlanCmds,
	})

	// Step 2: Create SVI with IP (if gateway provided)
	if p.Gateway != "" && p.Netmask != "" {
		sviCmds := []string{fmt.Sprintf("interface Vlan%d", p.VlanID)}
		if p.VRF != "" {
			sviCmds = append(sviCmds, fmt.Sprintf("vrf member %s", p.VRF))
		}
		sviCmds = append(sviCmds, fmt.Sprintf("ip address %s %s", p.Gateway, p.Netmask), "no shutdown", "exit")
		steps = append(steps, vlanProvisionStep{
			Action:   "create_svi",
			Commands: sviCmds,
		})
	}

	// Save config
	steps = append(steps, vlanProvisionStep{
		Action:   "save_config",
		Commands: []string{"end", "copy running-config startup-config"},
	})

	return steps
}

func generateIOSVLANProvision(p *SwitchVLANProvisionPayload) []vlanProvisionStep {
	var steps []vlanProvisionStep

	// Step 1: Create VLAN
	vlanCmds := []string{"configure terminal", fmt.Sprintf("vlan %d", p.VlanID)}
	if p.VlanName != "" {
		vlanCmds = append(vlanCmds, fmt.Sprintf("name %s", p.VlanName))
	}
	vlanCmds = append(vlanCmds, "exit")
	steps = append(steps, vlanProvisionStep{
		Action:   "create_vlan",
		Commands: vlanCmds,
	})

	// Step 2: Create SVI with IP (if gateway provided)
	if p.Gateway != "" && p.Netmask != "" {
		sviCmds := []string{fmt.Sprintf("interface Vlan%d", p.VlanID)}
		if p.VRF != "" {
			sviCmds = append(sviCmds, fmt.Sprintf("vrf forwarding %s", p.VRF))
		}
		sviCmds = append(sviCmds, fmt.Sprintf("ip address %s %s", p.Gateway, p.Netmask), "no shutdown", "exit")
		steps = append(steps, vlanProvisionStep{
			Action:   "create_svi",
			Commands: sviCmds,
		})
	}

	// Save config
	steps = append(steps, vlanProvisionStep{
		Action:   "save_config",
		Commands: []string{"end", "write memory"},
	})

	return steps
}

// netmaskToPrefixLen converts a dotted subnet mask to a prefix length.
func netmaskToPrefixLen(mask string) int {
	ip := net.ParseIP(mask)
	if ip == nil {
		return 24 // safe default
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return 24
	}
	ones, _ := net.IPv4Mask(ip4[0], ip4[1], ip4[2], ip4[3]).Size()
	return ones
}

// normalizeVendor maps various vendor name forms to a canonical key.
func normalizeVendor(vendor string) string {
	switch strings.ToLower(strings.TrimSpace(vendor)) {
	case "cisco", "cisco_ios":
		return "cisco_ios"
	case "cisco_nxos", "nxos", "nexus":
		return "cisco_nxos"
	case "juniper", "junos":
		return "junos"
	case "arista", "arista_eos":
		return "arista_eos"
	case "sonic":
		return "sonic"
	case "cumulus":
		return "cumulus"
	default:
		return strings.ToLower(vendor)
	}
}
