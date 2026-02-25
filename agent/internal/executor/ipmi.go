package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// ipmitoolPath caches the resolved path to ipmitool binary.
// On Linux, ipmitool is often at /usr/sbin/ipmitool which may not be
// in PATH for non-root users or certain process environments.
var (
	ipmitoolPath     string
	ipmitoolPathOnce sync.Once
)

func resolveIPMITool() string {
	// Try PATH first
	if p, err := exec.LookPath("ipmitool"); err == nil {
		return p
	}
	// Search common Linux installation paths
	for _, p := range []string{
		"/usr/sbin/ipmitool",
		"/usr/bin/ipmitool",
		"/usr/local/sbin/ipmitool",
		"/usr/local/bin/ipmitool",
		"/sbin/ipmitool",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Fallback — will produce a clear error if truly missing
	return "ipmitool"
}

func getIPMIToolPath() string {
	ipmitoolPathOnce.Do(func() {
		ipmitoolPath = resolveIPMITool()
	})
	return ipmitoolPath
}

// IPMIExecutor handles IPMI commands via ipmitool
type IPMIExecutor struct {
	logger *zap.Logger
}

func NewIPMIExecutor(logger *zap.Logger) *IPMIExecutor {
	ipmitoolBin := getIPMIToolPath()
	logger.Info("ipmitool resolved", zap.String("path", ipmitoolBin))
	return &IPMIExecutor{logger: logger}
}

type PowerPayload struct {
	IPMIIP   string `json:"ipmi_ip"`
	IPMIUser string `json:"ipmi_user"`
	IPMIPass string `json:"ipmi_pass"`
	BMCType  string `json:"bmc_type"`
}

func (e *IPMIExecutor) runIPMI(bmcType, ip, user, pass string, args ...string) (string, error) {
	// Strip CIDR notation if present (e.g. "10.0.0.1/32" → "10.0.0.1")
	if idx := strings.IndexByte(ip, '/'); idx != -1 {
		ip = ip[:idx]
	}

	cmdArgs := []string{"-I", "lanplus", "-H", ip, "-U", user, "-P", pass}

	// Vendor-specific ipmitool parameters
	switch bmcType {
	case "supermicro":
		// Some Supermicro models require cipher suite 17
		cmdArgs = append(cmdArgs, "-C", "17")
	case "huawei_ibmc":
		// Huawei iBMC often needs cipher suite 3
		cmdArgs = append(cmdArgs, "-C", "3")
	}

	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command(getIPMIToolPath(), cmdArgs...)

	e.logger.Debug("running ipmitool", zap.String("bmc_type", bmcType), zap.Strings("args", args), zap.String("host", ip))

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("ipmitool error: %s: %w", string(out), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (e *IPMIExecutor) parsePayload(raw json.RawMessage) (*PowerPayload, error) {
	var p PowerPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}
	if p.IPMIIP == "" || p.IPMIUser == "" || p.IPMIPass == "" {
		return nil, fmt.Errorf("ipmi_ip, ipmi_user, ipmi_pass are required")
	}
	return &p, nil
}

func (e *IPMIExecutor) HandlePowerOn(raw json.RawMessage) (interface{}, error) {
	p, err := e.parsePayload(raw)
	if err != nil {
		return nil, err
	}
	out, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "chassis", "power", "on")
	return map[string]string{"output": out}, err
}

func (e *IPMIExecutor) HandlePowerOff(raw json.RawMessage) (interface{}, error) {
	p, err := e.parsePayload(raw)
	if err != nil {
		return nil, err
	}
	out, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "chassis", "power", "off")
	return map[string]string{"output": out}, err
}

func (e *IPMIExecutor) HandlePowerReset(raw json.RawMessage) (interface{}, error) {
	p, err := e.parsePayload(raw)
	if err != nil {
		return nil, err
	}
	out, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "chassis", "power", "reset")
	return map[string]string{"output": out}, err
}

func (e *IPMIExecutor) HandlePowerCycle(raw json.RawMessage) (interface{}, error) {
	p, err := e.parsePayload(raw)
	if err != nil {
		return nil, err
	}
	out, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "chassis", "power", "cycle")
	return map[string]string{"output": out}, err
}

func (e *IPMIExecutor) HandlePowerStatus(raw json.RawMessage) (interface{}, error) {
	p, err := e.parsePayload(raw)
	if err != nil {
		return nil, err
	}
	out, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "chassis", "power", "status")
	if err != nil {
		return nil, err
	}

	status := "unknown"
	lower := strings.ToLower(out)
	if strings.Contains(lower, "on") {
		status = "on"
	} else if strings.Contains(lower, "off") {
		status = "off"
	}

	return map[string]string{"status": status, "raw": out}, nil
}

func (e *IPMIExecutor) HandleSensors(raw json.RawMessage) (interface{}, error) {
	p, err := e.parsePayload(raw)
	if err != nil {
		return nil, err
	}
	out, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "sdr", "type", "Temperature", "Fan", "Voltage")
	if err != nil {
		return nil, err
	}

	var sensors []map[string]string
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "|", 5)
		if len(parts) >= 5 {
			sensors = append(sensors, map[string]string{
				"name":   strings.TrimSpace(parts[0]),
				"value":  strings.TrimSpace(parts[1]),
				"status": strings.TrimSpace(parts[2]),
			})
		}
	}

	return map[string]interface{}{"sensors": sensors}, nil
}
