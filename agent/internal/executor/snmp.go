package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// SNMPExecutor handles SNMP polling of switch port traffic counters.
type SNMPExecutor struct {
	logger *zap.Logger
}

func NewSNMPExecutor(logger *zap.Logger) *SNMPExecutor {
	return &SNMPExecutor{logger: logger}
}

type SNMPPollPayload struct {
	SwitchIP      string `json:"switch_ip"`
	SNMPCommunity string `json:"snmp_community"`
	SNMPVersion   string `json:"snmp_version"`
	SSHUser       string `json:"ssh_user"`
	SSHPass       string `json:"ssh_pass"`
	SSHPort       int    `json:"ssh_port"`
	Vendor        string `json:"vendor"`
}

type PortTraffic struct {
	PortIndex  int    `json:"port_index"`
	PortName   string `json:"port_name"`
	InOctets   uint64 `json:"in_octets"`
	OutOctets  uint64 `json:"out_octets"`
	Speed      uint64 `json:"speed"`
	OperStatus string `json:"oper_status"`
	VlanID     int    `json:"vlan_id"`
}

// HandleSNMPPoll polls switch port counters via snmpwalk/snmpget.
func (e *SNMPExecutor) HandleSNMPPoll(raw json.RawMessage) (interface{}, error) {
	var p SNMPPollPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	e.logger.Info("polling SNMP counters",
		zap.String("switch_ip", p.SwitchIP),
		zap.String("community", p.SNMPCommunity),
	)

	version := "-v2c"
	if p.SNMPVersion == "v3" {
		version = "-v3"
	}

	// OIDs for interface counters
	oids := map[string]string{
		"ifDescr":      "1.3.6.1.2.1.2.2.1.2",
		"ifInOctets":   "1.3.6.1.2.1.2.2.1.10",
		"ifOutOctets":  "1.3.6.1.2.1.2.2.1.16",
		"ifSpeed":      "1.3.6.1.2.1.2.2.1.5",
		"ifOperStatus": "1.3.6.1.2.1.2.2.1.8",
	}

	results := make(map[string]map[int]string) // oid_name -> port_index -> value
	for name, oid := range oids {
		out, err := exec.Command("snmpwalk", version, "-c", p.SNMPCommunity, p.SwitchIP, oid).CombinedOutput()
		if err != nil {
			e.logger.Debug("snmpwalk failed", zap.String("oid", name), zap.Error(err))
			continue
		}
		results[name] = parseSNMPWalk(string(out))
	}

	// Poll VLAN data via SNMP strategies
	ifIndexToPvid := e.pollPortVLANs(version, p.SNMPCommunity, p.SwitchIP)

	// If SNMP VLAN discovery failed and SSH credentials are available, try SSH fallback
	if len(ifIndexToPvid) == 0 && p.SSHUser != "" && p.SSHPass != "" {
		e.logger.Info("SNMP VLAN discovery failed, trying SSH fallback",
			zap.String("vendor", p.Vendor), zap.String("switch_ip", p.SwitchIP))
		sshVlans := e.pollVLANsViaSSH(p.SwitchIP, p.SSHPort, p.SSHUser, p.SSHPass, p.Vendor, results["ifDescr"])
		if len(sshVlans) > 0 {
			ifIndexToPvid = sshVlans
		}
	}

	// Build port traffic data
	var ports []PortTraffic
	for idx, name := range results["ifDescr"] {
		pt := PortTraffic{
			PortIndex: idx,
			PortName:  name,
		}
		if v, ok := results["ifInOctets"][idx]; ok {
			pt.InOctets, _ = strconv.ParseUint(v, 10, 64)
		}
		if v, ok := results["ifOutOctets"][idx]; ok {
			pt.OutOctets, _ = strconv.ParseUint(v, 10, 64)
		}
		if v, ok := results["ifSpeed"][idx]; ok {
			pt.Speed, _ = strconv.ParseUint(v, 10, 64)
		}
		if v, ok := results["ifOperStatus"][idx]; ok {
			switch v {
			case "1":
				pt.OperStatus = "up"
			case "2":
				pt.OperStatus = "down"
			default:
				pt.OperStatus = "unknown"
			}
		}
		if vid, ok := ifIndexToPvid[idx]; ok {
			pt.VlanID = vid
		}
		ports = append(ports, pt)
	}

	return map[string]interface{}{
		"switch_ip": p.SwitchIP,
		"ports":     ports,
	}, nil
}

// pollPortVLANs discovers port VLAN assignments via multiple SNMP strategies.
// Strategy 1: Q-BRIDGE-MIB — dot1dBasePortIfIndex + dot1qPvid (standard)
// Strategy 2: Cisco vmVlan — CISCO-VLAN-MEMBERSHIP-MIB (Cisco IOS)
func (e *SNMPExecutor) pollPortVLANs(version, community, switchIP string) map[int]int {
	ifIndexToPvid := make(map[int]int)

	// --- Strategy 1: Q-BRIDGE-MIB (dot1qPvid) ---
	// Build bridgePort → ifIndex mapping (may fail on some devices, that's OK)
	bridgeToIfIndex := make(map[int]int)
	bridgeOut, err := exec.Command("snmpwalk", version, "-c", community, switchIP, "1.3.6.1.2.1.17.1.4.1.2").CombinedOutput()
	if err == nil {
		for bp, val := range parseSNMPWalk(string(bridgeOut)) {
			ifIdx, _ := strconv.Atoi(val)
			if ifIdx > 0 {
				bridgeToIfIndex[bp] = ifIdx
			}
		}
		e.logger.Debug("dot1dBasePortIfIndex", zap.Int("count", len(bridgeToIfIndex)))
	} else {
		e.logger.Info("dot1dBasePortIfIndex not available, will try direct ifIndex mapping", zap.Error(err))
	}

	// dot1qPvid: bridgePort (or ifIndex) → PVID
	pvidOut, err := exec.Command("snmpwalk", version, "-c", community, switchIP, "1.3.6.1.2.1.17.7.1.4.5.1.1").CombinedOutput()
	if err == nil {
		for bp, val := range parseSNMPWalk(string(pvidOut)) {
			pvid, _ := strconv.Atoi(val)
			if pvid > 0 {
				if ifIdx, ok := bridgeToIfIndex[bp]; ok {
					ifIndexToPvid[ifIdx] = pvid
				} else {
					// Device indexes dot1qPvid by ifIndex directly
					ifIndexToPvid[bp] = pvid
				}
			}
		}
	} else {
		e.logger.Info("dot1qPvid not available", zap.Error(err))
	}

	if len(ifIndexToPvid) > 0 {
		e.logger.Info("polled port VLANs via Q-BRIDGE-MIB", zap.Int("count", len(ifIndexToPvid)))
		return ifIndexToPvid
	}

	// --- Strategy 2: Cisco vmVlan (CISCO-VLAN-MEMBERSHIP-MIB) ---
	// OID 1.3.6.1.4.1.9.9.68.1.2.2.1.2 — indexed by ifIndex directly
	vmOut, err := exec.Command("snmpwalk", version, "-c", community, switchIP, "1.3.6.1.4.1.9.9.68.1.2.2.1.2").CombinedOutput()
	if err == nil {
		for ifIdx, val := range parseSNMPWalk(string(vmOut)) {
			vid, _ := strconv.Atoi(val)
			if vid > 0 {
				ifIndexToPvid[ifIdx] = vid
			}
		}
		if len(ifIndexToPvid) > 0 {
			e.logger.Info("polled port VLANs via Cisco vmVlan", zap.Int("count", len(ifIndexToPvid)))
			return ifIndexToPvid
		}
	} else {
		e.logger.Debug("Cisco vmVlan not available", zap.Error(err))
	}

	e.logger.Warn("no VLAN data discovered via SNMP", zap.String("switch_ip", switchIP))
	return ifIndexToPvid
}

// pollVLANsViaSSH discovers VLAN-to-port mappings by SSHing into the switch and running CLI commands.
// Uses an interactive shell with PTY to send "terminal length 0" first, avoiding pagination.
// Returns ifIndex → VLAN ID mapping using port names from SNMP ifDescr to match.
func (e *SNMPExecutor) pollVLANsViaSSH(host string, port int, user, pass, vendor string, ifDescrMap map[int]string) map[int]int {
	if port == 0 {
		port = 22
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		e.logger.Warn("SSH VLAN fallback: dial failed", zap.Error(err))
		return nil
	}
	defer client.Close()

	// Choose commands based on vendor
	disablePager := "terminal length 0"
	showCmd := "show vlan brief"
	switch vendor {
	case "juniper":
		disablePager = "set cli screen-length 0"
		showCmd = "show vlans brief"
	case "sonic":
		disablePager = ""
		showCmd = "show vlan brief"
	case "cumulus":
		disablePager = ""
		showCmd = "net show bridge vlan"
	case "huawei":
		disablePager = "screen-length 0 temporary"
		showCmd = "display vlan brief"
	}

	output, err := e.sshShellExec(client, disablePager, showCmd)
	if err != nil {
		e.logger.Warn("SSH VLAN fallback: shell exec failed", zap.Error(err))
		return nil
	}

	e.logger.Info("SSH VLAN fallback: got output",
		zap.Int("bytes", len(output)),
		zap.Int("lines", len(strings.Split(output, "\n"))))

	// Build reverse map: normalized port name → ifIndex
	nameToIfIndex := make(map[string]int)
	for idx, name := range ifDescrMap {
		nameToIfIndex[normalizePortName(name)] = idx
	}

	// Parse "show vlan brief" output (Cisco IOS/NX-OS/Arista format)
	return parseShowVlanBrief(output, nameToIfIndex, e.logger)
}

// sshShellExec opens an interactive shell session with a PTY, sends "terminal length 0"
// to disable pagination, then runs the show command and collects the full output.
func (e *SNMPExecutor) sshShellExec(client *ssh.Client, disablePager, showCmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("session: %w", err)
	}
	defer session.Close()

	// Request PTY so the switch treats this as an interactive session
	if err := session.RequestPty("xterm", 24, 512, ssh.TerminalModes{
		ssh.ECHO: 0,
	}); err != nil {
		return "", fmt.Errorf("pty: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("stdin: %w", err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout: %w", err)
	}

	if err := session.Shell(); err != nil {
		return "", fmt.Errorf("shell: %w", err)
	}

	// Wait for initial prompt
	time.Sleep(1 * time.Second)

	// Disable pagination
	if disablePager != "" {
		fmt.Fprintf(stdin, "%s\n", disablePager)
		time.Sleep(500 * time.Millisecond)
	}

	// Use a unique marker to know when the output is done
	marker := "###END_OF_OUTPUT###"
	fmt.Fprintf(stdin, "%s\n", showCmd)
	time.Sleep(500 * time.Millisecond)
	fmt.Fprintf(stdin, "echo %s\n", marker)
	// Also try just printing the marker for non-Linux switches
	fmt.Fprintf(stdin, "\n")
	time.Sleep(500 * time.Millisecond)

	// Send exit to close the shell
	fmt.Fprintf(stdin, "exit\n")
	stdin.Close()

	// Read all output with a timeout
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		io.Copy(&buf, stdout)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		e.logger.Debug("SSH shell read timed out, using partial output")
	}

	// Wait for session to finish (ignore error from exit)
	_ = session.Wait()

	return buf.String(), nil
}

// parseShowVlanBrief parses "show vlan brief" output and maps ports to VLAN IDs.
// Handles Cisco NX-OS / IOS / Arista format:
//
//	VLAN Name                             Status    Ports
//	---- -------------------------------- --------- -------------------------------
//	1    default                          active    Eth1/1, Eth1/2
//	100  Production                       active    Eth1/3, Eth1/4
//	200  Development                      act/unsup
//
// Matches any VLAN status (active, act/unsup, act/lshut, suspend, etc.)
func parseShowVlanBrief(output string, nameToIfIndex map[string]int, logger *zap.Logger) map[int]int {
	result := make(map[int]int)

	// Match VLAN lines: VLAN_ID  name  status  [optional ports...]
	// Status can be: active, act/unsup, act/lshut, suspend, etc.
	vlanLineRe := regexp.MustCompile(`^\s*(\d+)\s+\S+\s+(?:active|act/\S+|suspend)\s*(.*)$`)

	// Continuation lines: heavily indented lines with port names
	contLineRe := regexp.MustCompile(`^\s{10,}(\S.*)$`)

	// Skip non-data lines (headers, separators, prompts, blank)
	skipLineRe := regexp.MustCompile(`^[\s-]*$|^VLAN\s+Name|^----`)

	var currentVlan int
	matchedVlans := 0
	for _, line := range strings.Split(output, "\n") {
		// Strip ANSI escape codes and carriage returns
		line = stripANSI(line)
		line = strings.TrimRight(line, "\r")

		if skipLineRe.MatchString(line) {
			continue
		}

		if m := vlanLineRe.FindStringSubmatch(line); m != nil {
			vid, _ := strconv.Atoi(m[1])
			currentVlan = vid
			matchedVlans++
			if ports := strings.TrimSpace(m[2]); ports != "" {
				mapPortsToVlan(ports, vid, nameToIfIndex, result)
			}
		} else if currentVlan > 0 {
			if m := contLineRe.FindStringSubmatch(line); m != nil {
				mapPortsToVlan(m[1], currentVlan, nameToIfIndex, result)
			} else {
				currentVlan = 0
			}
		}
	}

	logger.Info("parsed show vlan brief",
		zap.Int("vlans_found", matchedVlans),
		zap.Int("port_mappings", len(result)))
	return result
}

// stripANSI removes ANSI escape sequences from a string (common in PTY output).
func stripANSI(s string) string {
	ansiRe := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRe.ReplaceAllString(s, "")
}

// mapPortsToVlan parses a comma-separated port list and maps each to the VLAN ID.
func mapPortsToVlan(portList string, vlanID int, nameToIfIndex map[string]int, result map[int]int) {
	for _, p := range strings.Split(portList, ",") {
		pName := normalizePortName(strings.TrimSpace(p))
		if pName == "" {
			continue
		}
		if ifIdx, ok := nameToIfIndex[pName]; ok {
			result[ifIdx] = vlanID
		}
	}
}

// normalizePortName normalizes port names for comparison.
// Cisco uses various abbreviations: Eth1/1, Ethernet1/1, Gi0/1, GigabitEthernet0/1, etc.
func normalizePortName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)

	// Common Cisco NX-OS abbreviations
	replacements := []struct{ prefix, full string }{
		{"ethernet", "eth"},
		{"gigabitethernet", "gi"},
		{"fastethernet", "fa"},
		{"tengigabitethernet", "te"},
		{"twentyfivegige", "twe"},
		{"fortygigabitethernet", "fo"},
		{"hundredgige", "hu"},
		{"port-channel", "po"},
	}
	for _, r := range replacements {
		if strings.HasPrefix(name, r.prefix) {
			name = r.full + name[len(r.prefix):]
			break
		}
	}
	return name
}

// parseSNMPWalk parses snmpwalk output into port_index → value map.
// Format: iso.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet0/1"
func parseSNMPWalk(output string) map[int]string {
	result := make(map[int]string)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		// Extract index from OID
		oidPart := strings.TrimSpace(parts[0])
		lastDot := strings.LastIndex(oidPart, ".")
		if lastDot < 0 {
			continue
		}
		idxStr := oidPart[lastDot+1:]
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			continue
		}

		// Extract value
		valuePart := strings.TrimSpace(parts[1])
		// Remove type prefix like "STRING: ", "INTEGER: ", "Counter32: ", "Gauge32: "
		if colonIdx := strings.Index(valuePart, ":"); colonIdx >= 0 {
			valuePart = strings.TrimSpace(valuePart[colonIdx+1:])
		}
		// Remove quotes
		valuePart = strings.Trim(valuePart, "\"")
		result[idx] = valuePart
	}
	return result
}
