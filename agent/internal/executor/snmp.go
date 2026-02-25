package executor

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"go.uber.org/zap"
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

	// Poll VLAN data: dot1dBasePortIfIndex (bridge port -> ifIndex) and dot1qPvid (bridge port -> PVID)
	ifIndexToPvid := e.pollPortVLANs(version, p.SNMPCommunity, p.SwitchIP)

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

// pollPortVLANs queries dot1dBasePortIfIndex and dot1qPvid to build ifIndex → PVID mapping.
func (e *SNMPExecutor) pollPortVLANs(version, community, switchIP string) map[int]int {
	ifIndexToPvid := make(map[int]int)

	// dot1dBasePortIfIndex: bridgePort -> ifIndex
	bridgeOut, err := exec.Command("snmpwalk", version, "-c", community, switchIP, "1.3.6.1.2.1.17.1.4.1.2").CombinedOutput()
	if err != nil {
		e.logger.Debug("snmpwalk dot1dBasePortIfIndex failed", zap.Error(err))
		return ifIndexToPvid
	}
	bridgeToIfIndex := make(map[int]int)
	for bp, val := range parseSNMPWalk(string(bridgeOut)) {
		ifIdx, _ := strconv.Atoi(val)
		if ifIdx > 0 {
			bridgeToIfIndex[bp] = ifIdx
		}
	}

	// dot1qPvid: bridgePort -> PVID
	pvidOut, err := exec.Command("snmpwalk", version, "-c", community, switchIP, "1.3.6.1.2.1.17.7.1.4.5.1.1").CombinedOutput()
	if err != nil {
		e.logger.Debug("snmpwalk dot1qPvid failed", zap.Error(err))
		return ifIndexToPvid
	}
	for bp, val := range parseSNMPWalk(string(pvidOut)) {
		pvid, _ := strconv.Atoi(val)
		if pvid > 0 {
			if ifIdx, ok := bridgeToIfIndex[bp]; ok {
				ifIndexToPvid[ifIdx] = pvid
			} else {
				// Some devices index dot1qPvid by ifIndex directly
				ifIndexToPvid[bp] = pvid
			}
		}
	}

	e.logger.Debug("polled port VLANs", zap.Int("count", len(ifIndexToPvid)))
	return ifIndexToPvid
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
