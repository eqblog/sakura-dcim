package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"go.uber.org/zap"
)

// SOLExecutor handles IPMI Serial Over LAN sessions
type SOLExecutor struct {
	logger *zap.Logger
}

func NewSOLExecutor(logger *zap.Logger) *SOLExecutor {
	return &SOLExecutor{logger: logger}
}

type SOLPayload struct {
	IPMIIP   string `json:"ipmi_ip"`
	IPMIUser string `json:"ipmi_user"`
	IPMIPass string `json:"ipmi_pass"`
	Action   string `json:"action"` // activate, deactivate, info
}

func (e *SOLExecutor) HandleSOL(raw json.RawMessage) (interface{}, error) {
	var p SOLPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse SOL payload: %w", err)
	}
	if p.IPMIIP == "" || p.IPMIUser == "" || p.IPMIPass == "" {
		return nil, fmt.Errorf("ipmi_ip, ipmi_user, ipmi_pass are required")
	}

	switch p.Action {
	case "info":
		return e.solInfo(p)
	case "activate":
		return e.solActivate(p)
	case "deactivate":
		return e.solDeactivate(p)
	default:
		return nil, fmt.Errorf("unknown SOL action: %s", p.Action)
	}
}

func (e *SOLExecutor) solInfo(p SOLPayload) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ipmitool", "-I", "lanplus",
		"-H", p.IPMIIP, "-U", p.IPMIUser, "-P", p.IPMIPass,
		"sol", "info")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sol info: %s: %w", string(out), err)
	}

	e.logger.Debug("SOL info retrieved", zap.String("host", p.IPMIIP))
	return map[string]string{"output": string(out)}, nil
}

func (e *SOLExecutor) solActivate(p SOLPayload) (interface{}, error) {
	// First deactivate any existing session
	e.solDeactivate(p)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ipmitool", "-I", "lanplus",
		"-H", p.IPMIIP, "-U", p.IPMIUser, "-P", p.IPMIPass,
		"sol", "activate")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sol activate: %s: %w", string(out), err)
	}

	e.logger.Info("SOL session activated", zap.String("host", p.IPMIIP))
	return map[string]string{"status": "activated", "output": string(out)}, nil
}

func (e *SOLExecutor) solDeactivate(p SOLPayload) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ipmitool", "-I", "lanplus",
		"-H", p.IPMIIP, "-U", p.IPMIUser, "-P", p.IPMIPass,
		"sol", "deactivate")

	out, err := cmd.CombinedOutput()
	if err != nil {
		e.logger.Debug("sol deactivate (may be expected)", zap.String("output", string(out)))
		return map[string]string{"status": "deactivated", "output": string(out)}, nil
	}

	e.logger.Info("SOL session deactivated", zap.String("host", p.IPMIIP))
	return map[string]string{"status": "deactivated", "output": string(out)}, nil
}
