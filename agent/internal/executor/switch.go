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
	SwitchIP    string `json:"switch_ip"`
	SSHUser     string `json:"ssh_user"`
	SSHPass     string `json:"ssh_pass"`
	SSHPort     int    `json:"ssh_port"`
	Vendor      string `json:"vendor"`
	PortName    string `json:"port_name"`
	VlanID      int    `json:"vlan_id"`
	SpeedMbps   int    `json:"speed_mbps"`
	AdminStatus string `json:"admin_status"`
	Description string `json:"description"`
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
		if p.VlanID > 0 {
			cmds = append(cmds, "switchport mode access", fmt.Sprintf("switchport access vlan %d", p.VlanID))
		}
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
		if p.VlanID > 0 {
			cmds = append(cmds, "switchport", "switchport mode access", fmt.Sprintf("switchport access vlan %d", p.VlanID))
		}
		if p.SpeedMbps > 0 {
			// NX-OS uses speed in Mbps directly for 1G/10G/25G/40G/100G
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
		if p.VlanID > 0 {
			cmds = append(cmds, fmt.Sprintf("set interfaces %s unit 0 family ethernet-switching vlan members vlan%d", p.PortName, p.VlanID))
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
		if p.VlanID > 0 {
			cmds = append(cmds, "switchport mode access", fmt.Sprintf("switchport access vlan %d", p.VlanID))
		}
		if p.AdminStatus == "down" {
			cmds = append(cmds, "shutdown")
		} else {
			cmds = append(cmds, "no shutdown")
		}
		cmds = append(cmds, "end", "write memory")
		return cmds

	case "sonic":
		cmds := []string{}
		if p.VlanID > 0 {
			cmds = append(cmds, fmt.Sprintf("sudo config vlan member add %d %s", p.VlanID, p.PortName))
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
		if p.VlanID > 0 {
			cmds = append(cmds, fmt.Sprintf("net add bridge bridge ports %s", p.PortName))
			cmds = append(cmds, fmt.Sprintf("net add interface %s bridge access %d", p.PortName, p.VlanID))
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
