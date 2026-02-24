package executor

import (
	"encoding/json"
	"fmt"
	"net"
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
	switch strings.ToLower(vendor) {
	case "cisco", "cisco_ios":
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

	case "juniper", "junos":
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

	case "arista", "arista_eos":
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

	case "sonic", "cumulus":
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

	default:
		return nil
	}
}

// generateStatusCommands creates vendor-specific CLI commands for port status queries.
func generateStatusCommands(vendor, portName string) []string {
	switch strings.ToLower(vendor) {
	case "cisco", "cisco_ios":
		return []string{fmt.Sprintf("show interface %s", portName)}
	case "juniper", "junos":
		return []string{fmt.Sprintf("show interfaces %s", portName)}
	case "arista", "arista_eos":
		return []string{fmt.Sprintf("show interfaces %s", portName)}
	case "sonic":
		return []string{fmt.Sprintf("show interfaces status %s", portName)}
	case "cumulus":
		return []string{fmt.Sprintf("net show interface %s", portName)}
	default:
		return nil
	}
}
