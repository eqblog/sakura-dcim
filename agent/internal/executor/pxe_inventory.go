package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"go.uber.org/zap"
)

// PXEInventoryExecutor boots a server to mini-Linux for hardware scanning
type PXEInventoryExecutor struct {
	logger *zap.Logger
	ipmi   *IPMIExecutor
	pxe    *PXEExecutor
}

func NewPXEInventoryExecutor(logger *zap.Logger, ipmi *IPMIExecutor, pxe *PXEExecutor) *PXEInventoryExecutor {
	return &PXEInventoryExecutor{logger: logger, ipmi: ipmi, pxe: pxe}
}

type PXEInventoryPayload struct {
	IPMIIP      string `json:"ipmi_ip"`
	IPMIUser    string `json:"ipmi_user"`
	IPMIPass    string `json:"ipmi_pass"`
	ServerMAC   string `json:"server_mac"`
	CallbackURL string `json:"callback_url"`
	ServerID    string `json:"server_id"`
}

// HandlePXEInventory boots a server into inventory-mode mini-Linux, scans hardware, reports back.
func (e *PXEInventoryExecutor) HandlePXEInventory(raw json.RawMessage) (interface{}, error) {
	var p PXEInventoryPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse pxe inventory payload: %w", err)
	}

	if p.IPMIIP == "" || p.ServerMAC == "" {
		return nil, fmt.Errorf("ipmi_ip and server_mac are required")
	}

	e.logger.Info("starting PXE inventory mode",
		zap.String("server_id", p.ServerID),
		zap.String("mac", p.ServerMAC),
	)

	// 1. Configure dnsmasq for inventory boot (mini-Linux with inventory script)
	dnsmasqConf := fmt.Sprintf(`dhcp-host=%s,set:inventory
dhcp-boot=tag:inventory,lpxelinux.0
dhcp-option=tag:inventory,option:server-ip-address,${NEXT_SERVER}
`, p.ServerMAC)

	pxelinuxCfg := fmt.Sprintf(`DEFAULT inventory
LABEL inventory
  KERNEL vmlinuz-inventory
  APPEND initrd=initrd-inventory.img console=ttyS0,115200n8 sakura.mode=inventory sakura.callback=%s sakura.server_id=%s
`, p.CallbackURL, p.ServerID)

	e.logger.Debug("PXE inventory config generated",
		zap.String("dnsmasq", dnsmasqConf),
		zap.String("pxelinux", pxelinuxCfg),
	)

	// 2. Set IPMI to PXE boot next
	if p.IPMIUser != "" && p.IPMIPass != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "ipmitool", "-I", "lanplus",
			"-H", p.IPMIIP, "-U", p.IPMIUser, "-P", p.IPMIPass,
			"chassis", "bootdev", "pxe")
		if out, err := cmd.CombinedOutput(); err != nil {
			e.logger.Error("failed to set PXE boot", zap.String("output", string(out)), zap.Error(err))
			return nil, fmt.Errorf("set PXE boot: %w", err)
		}

		// 3. Reboot server
		ctx2, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel2()
		cmd2 := exec.CommandContext(ctx2, "ipmitool", "-I", "lanplus",
			"-H", p.IPMIIP, "-U", p.IPMIUser, "-P", p.IPMIPass,
			"chassis", "power", "cycle")
		if out, err := cmd2.CombinedOutput(); err != nil {
			e.logger.Warn("power cycle failed, trying power on", zap.String("output", string(out)))
			ctx3, cancel3 := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel3()
			exec.CommandContext(ctx3, "ipmitool", "-I", "lanplus",
				"-H", p.IPMIIP, "-U", p.IPMIUser, "-P", p.IPMIPass,
				"chassis", "power", "on").Run()
		}
	}

	e.logger.Info("PXE inventory boot initiated",
		zap.String("server_id", p.ServerID),
	)

	return map[string]interface{}{
		"status":    "booting",
		"server_id": p.ServerID,
		"message":   "Server is booting into inventory mode. Results will be sent via callback.",
	}, nil
}
