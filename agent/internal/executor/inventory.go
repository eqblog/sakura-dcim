package executor

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"go.uber.org/zap"
)

// InventoryExecutor handles hardware inventory detection
type InventoryExecutor struct {
	logger *zap.Logger
}

func NewInventoryExecutor(logger *zap.Logger) *InventoryExecutor {
	return &InventoryExecutor{logger: logger}
}

func (e *InventoryExecutor) HandleScan(raw json.RawMessage) (interface{}, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("inventory scan only supported on Linux")
	}

	result := map[string]interface{}{}

	// CPU info
	if out, err := exec.Command("lscpu").Output(); err == nil {
		cpuInfo := map[string]string{}
		for _, line := range strings.Split(string(out), "\n") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cpuInfo[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
		result["cpu"] = cpuInfo
	}

	// Memory info
	if out, err := exec.Command("dmidecode", "-t", "memory").Output(); err == nil {
		result["memory_raw"] = string(out)
	}

	// Disk info
	if out, err := exec.Command("lsblk", "-J", "-o", "NAME,SIZE,TYPE,MODEL,SERIAL,ROTA,TRAN").Output(); err == nil {
		var diskInfo interface{}
		if json.Unmarshal(out, &diskInfo) == nil {
			result["disks"] = diskInfo
		}
	}

	// Network interfaces
	if out, err := exec.Command("ip", "-j", "addr", "show").Output(); err == nil {
		var netInfo interface{}
		if json.Unmarshal(out, &netInfo) == nil {
			result["network"] = netInfo
		}
	}

	// DMI system info
	if out, err := exec.Command("dmidecode", "-t", "system").Output(); err == nil {
		result["system_raw"] = string(out)
	}

	e.logger.Info("inventory scan completed", zap.Int("components", len(result)))
	return result, nil
}
