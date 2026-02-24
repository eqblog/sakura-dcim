package executor

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// IPMIExecutor handles IPMI commands via ipmitool
type IPMIExecutor struct {
	logger *zap.Logger
}

func NewIPMIExecutor(logger *zap.Logger) *IPMIExecutor {
	return &IPMIExecutor{logger: logger}
}

type PowerPayload struct {
	IPMIIP   string `json:"ipmi_ip"`
	IPMIUser string `json:"ipmi_user"`
	IPMIPass string `json:"ipmi_pass"`
}

func (e *IPMIExecutor) runIPMI(ip, user, pass string, args ...string) (string, error) {
	cmdArgs := append([]string{"-I", "lanplus", "-H", ip, "-U", user, "-P", pass}, args...)
	cmd := exec.Command("ipmitool", cmdArgs...)

	e.logger.Debug("running ipmitool", zap.Strings("args", args), zap.String("host", ip))

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
	out, err := e.runIPMI(p.IPMIIP, p.IPMIUser, p.IPMIPass, "chassis", "power", "on")
	return map[string]string{"output": out}, err
}

func (e *IPMIExecutor) HandlePowerOff(raw json.RawMessage) (interface{}, error) {
	p, err := e.parsePayload(raw)
	if err != nil {
		return nil, err
	}
	out, err := e.runIPMI(p.IPMIIP, p.IPMIUser, p.IPMIPass, "chassis", "power", "off")
	return map[string]string{"output": out}, err
}

func (e *IPMIExecutor) HandlePowerReset(raw json.RawMessage) (interface{}, error) {
	p, err := e.parsePayload(raw)
	if err != nil {
		return nil, err
	}
	out, err := e.runIPMI(p.IPMIIP, p.IPMIUser, p.IPMIPass, "chassis", "power", "reset")
	return map[string]string{"output": out}, err
}

func (e *IPMIExecutor) HandlePowerCycle(raw json.RawMessage) (interface{}, error) {
	p, err := e.parsePayload(raw)
	if err != nil {
		return nil, err
	}
	out, err := e.runIPMI(p.IPMIIP, p.IPMIUser, p.IPMIPass, "chassis", "power", "cycle")
	return map[string]string{"output": out}, err
}

func (e *IPMIExecutor) HandlePowerStatus(raw json.RawMessage) (interface{}, error) {
	p, err := e.parsePayload(raw)
	if err != nil {
		return nil, err
	}
	out, err := e.runIPMI(p.IPMIIP, p.IPMIUser, p.IPMIPass, "chassis", "power", "status")
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
	out, err := e.runIPMI(p.IPMIIP, p.IPMIUser, p.IPMIPass, "sdr", "type", "Temperature", "Fan", "Voltage")
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
